// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/bridge/protogo"
	"go.thesmos.sh/eidos/eidostest/protopipe"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// runProtogo drives the protobuf frontend against the named
// fixture, registers the protogo bridge as an annotator, and
// returns the resulting node packages. The helper centralises
// the harness wiring so every test stays focused on its
// assertion surface.
func runProtogo(t *testing.T, fixture string) []*node.Package {
	t.Helper()
	root := fixtureRoot(t, fixture)
	result := protopipe.Run(t, protopipe.RunOptions{
		SourceDir:  root,
		Annotators: []plugin.Annotator{protogo.New()},
	})
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	var pkgs []*node.Package
	result.Store.Nodes().Packages().Range(func(p *node.Package) bool {
		pkgs = append(pkgs, p)
		return true
	})
	return pkgs
}

// findPackage returns the package whose Path matches path; fails
// the test when no match. The error path enumerates the
// available paths so the failure message points the reader at
// the discrepancy directly.
func findPackage(t *testing.T, pkgs []*node.Package, path string) *node.Package {
	t.Helper()
	for _, p := range pkgs {
		if p.Path == path {
			return p
		}
	}
	paths := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		paths = append(paths, p.Path)
	}
	t.Fatalf("package %q missing; got %+v", path, paths)
	return nil
}

// findStruct returns the Struct whose Name matches name on pkg
// or nil when no match. The nil return lets callers surface a
// fixture-specific failure message rather than a generic one.
func findStruct(pkg *node.Package, name string) *node.Struct {
	for _, s := range pkg.Structs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// fixtureRoot resolves the absolute path of the
// frontend/protobuf/testdata/<name> directory through
// [runtime.Caller], so the resolved path is stable regardless
// of the test's working directory.
func fixtureRoot(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	return filepath.Join(repoRoot, "frontend", "protobuf", "testdata", name)
}

// collectSinkBody concatenates every written entry from mem
// into one string for grep-style assertions, separating files
// with a `--- <dir>/<filename> ---` banner so a failure message
// shows which file carried the offending content.
func collectSinkBody(mem *sink.Memory) string {
	var b strings.Builder
	for k, v := range mem.Files() {
		b.WriteString("--- ")
		b.WriteString(k.Dir)
		b.WriteByte('/')
		b.WriteString(k.Filename)
		b.WriteString(" ---\n")
		b.Write(v)
		b.WriteByte('\n')
	}
	return b.String()
}
