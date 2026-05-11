// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"slices"
	"testing"

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
