// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package docaudit drives the "documented vs implemented"
// audit every plugin package runs against its own `doc.go`.
// Each meta-key constructor literal under the package source
// must appear in the package doc; otherwise the audit fails and
// the offending key surfaces in the failing-test output.
//
// The audit ships as a black-box helper rather than per-package
// duplication so a single update to the key-discovery rule
// (new constructor name, additional skip pattern, ...)
// propagates without touching every doc-audit caller.
package docaudit

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// TB is the narrow subset of [testing.TB] the audit needs.
// Defined as a local interface so the audit accepts both
// real *testing.T values and per-test fakes that drive failure-
// path coverage on the helper itself.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// AssertEveryMetaKeyDocumented walks every `.go` source file
// under packageDir (excluding `_test.go`), extracts the literal-
// string first argument of every `meta.NewKey(...)` /
// `meta.EnsureKey(...)` call, and asserts each resulting key
// appears verbatim in packageDir/doc.go. Dynamic-name calls
// whose first argument is not a literal string are skipped;
// those land under a documented namespace prefix, which the
// caller audits separately via the prefix presence in doc.go.
//
// Pass the test's `t` and the absolute path of the package
// directory. The package's doc.go is read by joining packageDir
// with the literal filename.
func AssertEveryMetaKeyDocumented(t TB, packageDir string) {
	t.Helper()
	keys, err := collectMetaKeyLiterals(packageDir)
	if err != nil {
		t.Fatalf("docaudit: collect meta-key literals from %s: %v", packageDir, err)
	}
	if len(keys) == 0 {
		t.Fatalf("docaudit: no meta-key literals discovered under %s — audit is mis-wired",
			packageDir)
	}
	docPath := filepath.Join(packageDir, "doc.go")
	body, err := os.ReadFile(docPath) //nolint:gosec // path supplied by test code, not user input.
	if err != nil {
		t.Fatalf("docaudit: read %s: %v", docPath, err)
	}
	doc := string(body)
	for _, key := range keys {
		if !strings.Contains(doc, key) {
			t.Errorf("docaudit: doc.go missing mention of meta key %q (declared in package source)",
				key)
		}
	}
}

// collectMetaKeyLiterals returns the sorted, de-duplicated list
// of literal-string first arguments to every meta-key
// constructor call found under dir's non-test Go sources.
// Returns [ErrEmptyDirectory] when no source files parse.
func collectMetaKeyLiterals(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err //nolint:wrapcheck // pass-through for caller's t.Fatalf wrap
	}
	fset := token.NewFileSet()
	seen := map[string]struct{}{}
	parsedAny := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err //nolint:wrapcheck // pass-through for caller's t.Fatalf wrap
		}
		parsedAny = true
		ast.Inspect(file, func(n ast.Node) bool {
			literal, ok := metaKeyLiteralFrom(n)
			if !ok {
				return true
			}
			seen[literal] = struct{}{}
			return true
		})
	}
	if !parsedAny {
		return nil, ErrEmptyDirectory
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	slices.Sort(out)
	return out, nil
}

// metaKeyLiteralFrom returns the first-argument literal string
// of n when n is a `meta.NewKey(...)` or `meta.EnsureKey(...)`
// call; ok=false otherwise. Calls whose first argument is not a
// literal string (the dynamic-name pattern) return ok=false too
// — those are documented by namespace prefix, not per-key
// enumeration.
func metaKeyLiteralFrom(n ast.Node) (string, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "meta" {
		return "", false
	}
	if sel.Sel.Name != "NewKey" && sel.Sel.Name != "EnsureKey" {
		return "", false
	}
	if len(call.Args) == 0 {
		return "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := unquoteDoubleQuoted(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

// unquoteDoubleQuoted strips the surrounding double quotes from
// raw and returns the inner content. The meta-key constructors
// only ever take double-quoted literals in this codebase; the
// helper deliberately rejects raw-string and single-quoted forms
// to keep the audit's input contract narrow.
func unquoteDoubleQuoted(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", ErrUnsupportedLiteral
	}
	return raw[1 : len(raw)-1], nil
}

// ErrEmptyDirectory signals that a doc-audit walk under the
// supplied directory found no parseable Go source files. The
// audit treats the situation as a wiring error rather than an
// implicit pass.
var ErrEmptyDirectory = errors.New("docaudit: no Go source files in package directory")

// ErrUnsupportedLiteral signals that a meta-key constructor call
// carried a literal kind the audit's narrow grammar doesn't
// accept (raw string, single-quoted, multi-line backtick). The
// audit skips the call rather than guessing the key string.
var ErrUnsupportedLiteral = errors.New("docaudit: unsupported literal form")
