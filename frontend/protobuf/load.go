// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/plugin"
)

// errEditionSource is the sentinel returned by [scanSyntax] when a
// proto source declares `edition = "..."`. The frontend supports
// `syntax = "proto3";` only; editions sources surface a positioned
// `diag.Error` and are skipped. Compared with [errors.Is].
//
//nolint:gochecknoglobals // sentinel error per the framework convention.
var errEditionSource = errors.New("protobuf: edition source detected")

// loadPattern is the per-pattern Load entry point invoked from
// [Frontend.Load]. The resolved descriptor set composes into the
// per-plugin cache key and the source-mapping converter walks it
// into [store.Store.Nodes] entries. An empty Pattern is a no-op
// so plugin-shape tests that don't supply a proto source root
// drive the frontend without hitting the compiler.
func loadPattern(ctx *plugin.FrontendContext, opts Options) error {
	if ctx == nil || ctx.Pattern == "" {
		return nil
	}
	ps := ctx.Diag.For(FrontendName)
	root := resolveRoot(opts.Dir)
	files, err := discoverProtoFiles(root, ctx.Pattern)
	if err != nil {
		ps.Errorf(position.Pos{}, "protobuf: discover %s in %s: %v", ctx.Pattern, root, err)
		return nil
	}
	entries := filterUnsupportedSyntax(ps, root, files)
	if len(entries) == 0 {
		return nil
	}
	compiler := newCompiler(ps, opts, root)
	resolved, err := compiler.Compile(context.Background(), entries...)
	switch {
	case err == nil:
		// Compile succeeded; the resolved descriptor set is
		// fully populated and safe to hash for the cache key.
	case errors.Is(err, reporter.ErrInvalidSource):
		// Parse / resolution errors already flowed through the
		// reporter callbacks attached to Handler; the returned
		// sentinel is the terminal indicator that the run hit at
		// least one such error, with the authoritative
		// per-position diagnostics already recorded on ps. The
		// returned descriptor slice is partial — entries for files
		// that failed to parse are nil — so the cache step is
		// skipped (there is no coherent descriptor set to hash).
		return nil
	default:
		// Any other error (context cancellation, internal panic
		// wrapped as error, resolver failure outside the reporter
		// path) needs a surface so consumers don't miss it.
		ps.Errorf(position.Pos{}, "protobuf: compile %s: %v", root, err)
		return nil
	}
	// Cache consultation: the frontend composes a content-addressed
	// key from the resolved descriptor set + its declared options
	// and version, consults the configured cache, and stores an
	// entry for the next run. The stored payload is the serialized
	// node-graph form once the converter populates it; consumers
	// reading the cache treat an empty body as the no-payload
	// sentinel and fall through to a fresh parse.
	descriptors := make([]protoreflect.FileDescriptor, 0, len(resolved))
	for _, f := range resolved {
		descriptors = append(descriptors, f)
	}
	key := composeCacheKey(ps, opts, descriptors)
	if _, hit := consultCache(ctx.Cache, key); !hit {
		storeCache(ctx.Cache, ps, key, nil)
	}
	convertFiles(ctx, ps, descriptors)
	return nil
}

// resolveRoot returns the directory the resolver searches for
// proto sources. Falls back to the process working directory when
// the configured [Options.Dir] is empty (the documented
// zero-config behaviour).
func resolveRoot(configured string) string {
	if configured != "" {
		return configured
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

// discoverProtoFiles walks root and returns every `.proto` file
// reachable per pattern. Two pattern forms are supported:
//
//   - `./...` — recursive walk for every `.proto` under root.
//   - `<file.proto>` — a single literal file path relative to root.
//
// Patterns other than `./...` are interpreted as single proto
// file paths relative to root; subdirectory globs and shell
// wildcards are not expanded. Callers wanting a narrower scope
// than `./...` either run the frontend per-file or extend this
// surface with explicit glob handling.
//
// Returned paths are root-relative so protocompile's resolver
// finds them via the configured [SourceResolver.ImportPaths]. The
// slice is sorted alphabetically for deterministic compile order.
func discoverProtoFiles(root, pattern string) ([]string, error) {
	if pattern != "./..." {
		return []string{pattern}, nil
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %s: %w", path, walkErr)
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".proto" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return fmt.Errorf("rel %s: %w", path, relErr)
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("protobuf: walk %s: %w", root, err)
	}
	sort.Strings(out)
	return out, nil
}

// filterUnsupportedSyntax pre-scans each file for the proto3-only
// guard. Files whose first declaration is `edition = "..."`
// surface a positioned `diag.Error` and are dropped from the
// compile set; remaining files continue to the protocompile
// compile pass. proto2 sources reach protocompile and surface as
// resolution errors via the standard reporter callback —
// protocompile handles proto2's syntax declaration on the
// resolver path; the editions guard here is the spec-mandated
// fast-path because editions is the proto3-incompatible case
// users most commonly write.
func filterUnsupportedSyntax(ps *diag.PluginSink, root string, files []string) []string {
	out := make([]string, 0, len(files))
	for _, rel := range files {
		abs := filepath.Join(root, rel)
		switch err := scanSyntax(abs); {
		case errors.Is(err, errEditionSource):
			ps.Errorf(
				position.Pos{File: abs, Line: 1, Column: 1},
				"protobuf: %s declares an edition source (`edition = ...`); only proto3 is supported, file skipped",
				rel,
			)
		default:
			out = append(out, rel)
		}
	}
	return out
}

// scanSyntax reads the first non-blank, non-comment line of path
// and returns [errEditionSource] when it matches the `edition =
// "..."` syntax declaration the proto editions migration path
// introduces. Returns nil for files where the syntax declaration
// is `syntax = "proto3";`, missing, proto2, or anything else;
// protocompile rejects unsupported syntaxes through its own
// resolver path so the scanner only handles the editions case
// that needs the explicit guard.
//
// The match is identifier-bounded: a line starting with
// `edition` followed by whitespace or `=` triggers the guard;
// `edition_foo` or any other identifier starting with the
// substring does not. The scanner reads at most the first few
// non-blank lines so it stays cheap on large source files.
// Errors opening or reading the file return nil — the subsequent
// protocompile pass surfaces a positioned diagnostic against the
// same file, so a scanner-side error suppression here avoids
// duplicate messages.
func scanSyntax(path string) error {
	f, err := os.Open(path) //nolint:gosec // path is from the configured search root.
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}
		if isEditionDecl(line) {
			return errEditionSource
		}
		// Any other first-non-comment line means the file isn't an
		// edition source; let protocompile handle it.
		return nil
	}
	return nil
}

// isEditionDecl reports whether line is a proto `edition`
// declaration — `edition` followed by whitespace or `=`. Tight
// match so `edition_foo` and similar identifiers starting with
// the substring don't fire the editions guard.
func isEditionDecl(line string) bool {
	const keyword = "edition"
	if !strings.HasPrefix(line, keyword) {
		return false
	}
	rest := line[len(keyword):]
	if rest == "" {
		return true
	}
	next := rest[0]
	return next == ' ' || next == '\t' || next == '='
}

// newCompiler constructs the protocompile.Compiler configured for
// the supplied options. The reporter callbacks funnel errors and
// warnings through the frontend's per-plugin diagnostic sink so
// every diagnostic is attributed to the protobuf frontend's name
// with positions resolved through protocompile.
func newCompiler(ps *diag.PluginSink, opts Options, root string) *protocompile.Compiler {
	extra := importPathList(opts.ImportPaths)
	roots := make([]string, 0, 1+len(extra))
	roots = append(roots, root)
	roots = append(roots, extra...)
	resolver := protocompile.Resolver(&protocompile.SourceResolver{ImportPaths: roots})
	if opts.IncludeWellKnown {
		resolver = protocompile.WithStandardImports(resolver)
	}
	rep := reporter.NewReporter(
		func(err reporter.ErrorWithPos) error {
			ps.Errorf(posFromReporter(err), "protobuf: %s", err.Unwrap())
			// Returning nil here keeps protocompile compiling the
			// remaining files rather than aborting on the first
			// error — the framework's run-to-completion contract
			// demands the per-file diagnostics surface together so
			// users see every parse error in one run.
			return nil
		},
		func(warn reporter.ErrorWithPos) {
			ps.Warnf(posFromReporter(warn), "protobuf: %s", warn.Unwrap())
		},
	)
	return &protocompile.Compiler{Resolver: resolver, Reporter: rep}
}

// posFromReporter translates protocompile's per-error source
// position into the framework's [position.Pos] form. protocompile
// reports file + line + column tuples; the framework's `Offset`
// field stays zero per the spec's Position translation rule.
func posFromReporter(err reporter.ErrorWithPos) position.Pos {
	p := err.GetPosition()
	return position.Pos{File: p.Filename, Line: p.Line, Column: p.Col}
}
