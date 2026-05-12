// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/manifest"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a Manifest stamped with current Version and supplied RunID", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		if m.Version != manifest.Version {
			t.Fatalf("Version = %d, want %d", m.Version, manifest.Version)
		}
		if m.RunID != "run-1" {
			t.Fatalf("RunID = %q, want run-1", m.RunID)
		}
		if len(m.Outputs) != 0 {
			t.Fatalf("new Manifest should have no outputs; got %d", len(m.Outputs))
		}
		if m.Brand != "" {
			t.Fatalf("Brand should default to empty for the caller to stamp; got %q", m.Brand)
		}
	})

	t.Run("Brand is a stamped field callers populate after construction", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-2")
		m.Brand = "acmegen"
		if m.Brand != "acmegen" {
			t.Fatalf("Brand mutation didn't stick; got %q", m.Brand)
		}
	})
}

func TestManifest_Add(t *testing.T) {
	t.Parallel()

	t.Run("appends an output entry", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		m.Add(manifest.Output{Target: targetAt("a", "b.go"), Plugins: []string{"p"}, Hash: "sha256:x"})
		if len(m.Outputs) != 1 {
			t.Fatalf("Add should append; Outputs=%d", len(m.Outputs))
		}
	})
}

func TestManifest_Targets(t *testing.T) {
	t.Parallel()

	t.Run("returns the target list in append order", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		m.Add(manifest.Output{Target: targetAt("a", "first.go")})
		m.Add(manifest.Output{Target: targetAt("a", "second.go")})
		got := m.Targets()
		want := []string{"first.go", "second.go"}
		gotNames := []string{got[0].Filename, got[1].Filename}
		if !slices.Equal(gotNames, want) {
			t.Fatalf("Targets order mismatch: %v", gotNames)
		}
	})

	t.Run("returns an empty slice for an empty manifest", func(t *testing.T) {
		t.Parallel()
		got := manifest.New("run-1").Targets()
		if len(got) != 0 {
			t.Fatalf("empty manifest should yield no targets; got %d", len(got))
		}
	})
}

func TestManifest_HasTarget(t *testing.T) {
	t.Parallel()

	t.Run("returns true for a recorded target", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		m.Add(manifest.Output{Target: targetAt("a", "b.go")})
		if !m.HasTarget(targetAt("a", "b.go")) {
			t.Fatalf("HasTarget should return true for recorded target")
		}
	})

	t.Run("returns false for an unknown target", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		if m.HasTarget(targetAt("a", "missing.go")) {
			t.Fatalf("HasTarget should return false for unknown target")
		}
	})
}
