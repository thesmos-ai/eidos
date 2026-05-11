// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"go/format"
	"go/parser"
	"go/token"
	"strconv"

	"golang.org/x/tools/imports"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/writer"
)

// finaliseBody runs body through the canonical Go finalisation
// chain: [go/format.Source] for gofmt-clean formatting, then the
// goimports library pass for stdlib/external regrouping and import
// hygiene. Both steps' failures are non-fatal — they attach a Warn
// to ps and fall back to the best-available bytes — so a content
// defect never aborts the surrounding render loop.
//
// tracked is the set of import paths the backend recorded during
// template rendering. After goimports completes, any import in the
// finalised output that does not appear in tracked surfaces as one
// Warn per uncovered path — flagging a converter or template bug
// that left an unrecorded type reference behind.
func finaliseBody(body []byte, target emit.Target, ps *diag.PluginSink, tracked []writer.Import) []byte {
	formatted, ok := runGoFormat(body, target, ps)
	if !ok {
		return body
	}
	return runGoImports(formatted, target, ps, trackedPaths(tracked))
}

// runGoFormat runs body through [go/format.Source]. Returns the
// formatted bytes and ok=true on success; on failure attaches a
// Warn diagnostic and returns (body, false) so the caller falls
// back to the unformatted bytes.
func runGoFormat(body []byte, target emit.Target, ps *diag.PluginSink) ([]byte, bool) {
	out, err := format.Source(body)
	if err != nil {
		ps.Warnf(position.Pos{}, "%s: format.Source failed: %v", target.JoinPath(), err)
		return body, false
	}
	return out, true
}

// runGoImports runs the goimports library pass over body. On
// success, untracked imports that goimports added produce one Warn
// per path. On failure, attaches a Warn and falls back to body.
func runGoImports(body []byte, target emit.Target, ps *diag.PluginSink, tracked map[string]struct{}) []byte {
	out, err := imports.Process(target.JoinPath(), body, &imports.Options{
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false,
	})
	if err != nil {
		ps.Warnf(position.Pos{}, "%s: goimports failed: %v", target.JoinPath(), err)
		return body
	}
	for _, path := range importPaths(out, target.JoinPath()) {
		if _, recorded := tracked[path]; !recorded {
			ps.Warnf(position.Pos{}, "%s: goimports added untracked import %q", target.JoinPath(), path)
		}
	}
	return out
}

// trackedPaths builds a set of import paths from the slice of
// [writer.Import] the backend recorded. The set form makes the
// "is this path tracked?" lookup in [runGoImports] O(1).
func trackedPaths(imports []writer.Import) map[string]struct{} {
	out := make(map[string]struct{}, len(imports))
	for _, imp := range imports {
		out[imp.Path] = struct{}{}
	}
	return out
}

// importPaths parses src in imports-only mode and returns each
// import declaration's path with surrounding quotes stripped. Parse
// errors return an empty slice — callers degrade gracefully.
func importPaths(src []byte, filename string) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		out = append(out, path)
	}
	return out
}
