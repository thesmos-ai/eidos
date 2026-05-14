// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"path/filepath"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/frontend/golang"
)

func TestFrontendName(t *testing.T) {
	t.Parallel()

	t.Run("matches the canonical plugin name", func(t *testing.T) {
		t.Parallel()
		if golang.FrontendName != "golang" {
			t.Fatalf("FrontendName = %q, want %q", golang.FrontendName, "golang")
		}
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil Frontend", func(t *testing.T) {
		t.Parallel()
		if golang.New() == nil {
			t.Fatalf("New returned nil")
		}
	})
}

func TestFrontend_Name(t *testing.T) {
	t.Parallel()

	t.Run("returns FrontendName", func(t *testing.T) {
		t.Parallel()
		if got := golang.New().Name(); got != golang.FrontendName {
			t.Fatalf("Name = %q, want %q", got, golang.FrontendName)
		}
	})
}

func TestFrontend_Version(t *testing.T) {
	t.Parallel()

	t.Run("returns FrontendVersion", func(t *testing.T) {
		t.Parallel()
		if got := golang.New().Version(); got != golang.FrontendVersion {
			t.Fatalf("Version = %q, want %q", got, golang.FrontendVersion)
		}
	})
}

func TestFrontend_EmitVersions(t *testing.T) {
	t.Parallel()

	t.Run("contains the in-tree emit major", func(t *testing.T) {
		t.Parallel()
		majors := golang.New().EmitVersions()
		want := emit.Major()
		if !slices.Contains(majors, want) {
			t.Fatalf("EmitVersions = %v, expected to include %q", majors, want)
		}
	})

	t.Run("returns an independent copy", func(t *testing.T) {
		t.Parallel()
		fe := golang.New()
		first := fe.EmitVersions()
		if len(first) == 0 {
			t.Fatalf("EmitVersions must report at least one major")
		}
		first[0] = "tampered"
		second := fe.EmitVersions()
		if second[0] == "tampered" {
			t.Fatalf("EmitVersions returned an aliased slice; mutation leaked back into the frontend")
		}
	})
}

// TestConformance runs the framework's plugin-conformance suite
// against this package's plugin. The suite pins the standard
// framework contracts (stable Name, role-interface compliance,
// deterministic capability ordering, unique directive schema
// names, non-empty Versioned version) plus the per-role
// frontend contracts (empty-pattern panic recovery, determinism
// across two runs of the same source fixture).
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, golang.New())
	})

	t.Run("frontend contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunFrontendSuite(
			t,
			golang.New(),
			[]plugintest.FrontendFixture{
				{
					Name:    "basic_struct fixture",
					Pattern: "./...",
					Options: map[string]string{
						"dir": filepath.Join(goldenRoot, "basic_struct"),
					},
				},
				{
					Name:    "interface_with_methods fixture",
					Pattern: "./...",
					Options: map[string]string{
						"dir": filepath.Join(goldenRoot, "interface_with_methods"),
					},
				},
			},
		)
	})
}
