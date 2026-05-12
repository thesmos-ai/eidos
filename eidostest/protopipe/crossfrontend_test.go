// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protopipe_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/frontend/protobuf"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

// TestCrossFrontend_MarkerScope covers the cross-frontend
// provenance-marker convention: every produced [node.Package]
// carries the bare `frontend` meta key whose value is the
// producing frontend's plugin name. The test loads proto sources
// via the protobuf frontend and Go sources via the Go frontend
// into one shared store, then asserts:
//
//  1. Proto-derived packages carry `frontend = "protobuf"`.
//  2. Go-derived packages carry `frontend = "golang"`.
//  3. No `go.*` meta leaks onto proto-derived packages.
//  4. No `proto.*` meta leaks onto Go-derived packages.
//
// Together these assertions close the cross-namespace scope loop
// the bridge-annotator pattern relies on.
func TestCrossFrontend_MarkerScope(t *testing.T) {
	t.Parallel()

	t.Run("each frontend stamps its own marker; no cross-namespace meta leaks", func(t *testing.T) {
		t.Parallel()
		repoRoot := repositoryRoot(t)
		protoRoot := filepath.Join(repoRoot, "frontend", "protobuf", "testdata", "simple")
		s := store.New()
		d := diag.New()
		registry := directive.NewRegistry()
		parser := directive.DefaultParser()
		nopCache := cache.NewNone()

		pf := protobuf.New()
		if err := pf.SetOptions(opt.New(pf.OptionsSchema(), map[string]string{"dir": protoRoot})); err != nil {
			t.Fatalf("protobuf SetOptions: %v", err)
		}
		if err := pf.Load(&plugin.FrontendContext{
			Store: s, Diag: d, Registry: registry, Parser: parser, Cache: nopCache,
			Pattern: "./...",
		}); err != nil {
			t.Fatalf("protobuf Load: %v", err)
		}

		gf := golang.New()
		if err := gf.SetOptions(opt.New(gf.OptionsSchema(), map[string]string{"dir": repoRoot})); err != nil {
			t.Fatalf("golang SetOptions: %v", err)
		}
		if err := gf.Load(&plugin.FrontendContext{
			Store: s, Diag: d, Registry: registry, Parser: parser, Cache: nopCache,
			Pattern: "go.thesmos.sh/eidos/eidostest/pluginfixture",
		}); err != nil {
			t.Fatalf("golang Load: %v", err)
		}

		if d.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", d.Diagnostics())
		}

		var protoPkg, goPkg *node.Package
		s.Nodes().Packages().Range(func(p *node.Package) bool {
			switch p.Path {
			case "eidos.protobuf.testdata.simple":
				protoPkg = p
			case "go.thesmos.sh/eidos/eidostest/pluginfixture":
				goPkg = p
			}
			return true
		})
		if protoPkg == nil {
			t.Fatalf("proto-derived package missing from store")
		}
		if goPkg == nil {
			t.Fatalf("go-derived package missing from store")
		}

		assertMarker(t, protoPkg, protobuf.FrontendName)
		assertMarker(t, goPkg, golang.FrontendName)
		assertNoMetaPrefix(t, protoPkg, "go.", "proto-derived package")
		assertNoMetaPrefix(t, goPkg, "proto.", "go-derived package")
	})
}

// assertMarker verifies that pkg carries the cross-frontend marker
// with the expected value.
func assertMarker(t *testing.T, pkg *node.Package, want string) {
	t.Helper()
	got, ok := protobuf.MetaFrontend.Get(pkg.Meta())
	if !ok {
		t.Fatalf("MetaFrontend missing on package %q", pkg.Path)
	}
	if got != want {
		t.Fatalf("MetaFrontend on %q = %q, want %q", pkg.Path, got, want)
	}
}

// assertNoMetaPrefix walks pkg's meta bag and fails if any key
// matches prefix. The descriptor is the human-friendly label the
// test surfaces in the failure message.
func assertNoMetaPrefix(t *testing.T, pkg *node.Package, prefix, descriptor string) {
	t.Helper()
	for _, key := range pkg.Meta().Names() {
		if strings.HasPrefix(key, prefix) {
			t.Fatalf(
				"%s should carry no %q-namespaced meta; found %q on %q",
				descriptor, prefix, key, pkg.Path,
			)
		}
	}
}

// repositoryRoot resolves the absolute path of the eidos repo
// root. The test's working directory is the package directory by
// default; the helper walks up to the directory carrying go.mod
// so the cross-frontend pattern resolution finds the same module
// regardless of where `go test` is invoked from.
func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for dir != "/" {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("go.mod not found above %s", file)
	return ""
}
