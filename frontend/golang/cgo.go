// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"go/ast"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// IsCgoFile reports whether path identifies a cgo-synthesized Go
// file produced by [golang.org/x/tools/go/packages]. The match is
// deliberately permissive — false positives only mean a file that
// may or may not be cgo gets dropped under [Options.SkipCgoFiles],
// which a user can override by setting that option to false.
//
// Recognised conventions:
//
//   - basename "_cgo_gotypes.go" — the canonical go/packages cgo
//     output.
//   - basename prefixed with "_cgo_" — every other synthetic
//     output the cgo toolchain emits.
//   - any path containing "/cgo-gcc-prolog" — the cgo prologue
//     file inserted by the C compiler driver.
//
// Exported because the filter logic is a pure string predicate and
// black-box tests need a direct call site: the [packages.Load]
// machinery never surfaces cgo files in a fixture-only test
// (filenames starting with `_` are dropped by Go's build rules
// before our filter sees them), so the predicate itself is the
// only verifiable surface.
func IsCgoFile(path string) bool {
	if path == "" {
		return false
	}
	base := filepath.Base(path)
	if base == "_cgo_gotypes.go" {
		return true
	}
	if strings.HasPrefix(base, "_cgo_") {
		return true
	}
	if strings.Contains(path, "/cgo-gcc-prolog") {
		return true
	}
	return false
}

// filterSyntaxFiles returns the subset of pkg.Syntax the converter
// should walk. The default is every loaded syntax file; when
// [Options.SkipCgoFiles] is true (the default for production
// pipelines), cgo-synthesized files are dropped via [IsCgoFile] so
// generated declarations like the `_Ctype_*` wrappers don't pollute
// the store.
//
// The predicate's true branch is unreachable in pure-Go fixture
// tests because [golang.org/x/tools/go/packages] hides underscore-
// prefixed files before they appear in pkg.Syntax — [IsCgoFile] is
// the verifiable surface and carries its own black-box coverage.
func filterSyntaxFiles(pkg *packages.Package, opts Options) []*ast.File {
	return slices.DeleteFunc(slices.Clone(pkg.Syntax), func(f *ast.File) bool {
		return opts.SkipCgoFiles && IsCgoFile(pkg.Fset.Position(f.Pos()).Filename)
	})
}
