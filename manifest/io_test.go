// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/manifest"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("serialises a Manifest as JSON to the supplied path", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		path := filepath.Join(root, ".eidos", "manifest.json")
		m := manifest.New("run-1")
		m.Add(
			manifest.Output{
				Target:  targetAt("a", "b.go"),
				Plugins: []manifest.PluginAttribution{{Name: "p"}},
				Hash:    "sha256:x",
			},
		)
		assertNoError(t, manifest.Write(path, m))
		body, err := os.ReadFile(path)
		assertNoError(t, err)
		if !strings.Contains(string(body), "\"run_id\": \"run-1\"") {
			t.Fatalf("manifest body should include RunID; got %q", body)
		}
	})

	t.Run("creates intermediate directories under the destination", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		path := filepath.Join(root, "deep", "nested", "manifest.json")
		assertNoError(t, manifest.Write(path, manifest.New("run-1")))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected manifest at nested path; got %v", err)
		}
	})

	t.Run("rejects writing into a path whose parent is a regular file", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		conflict := filepath.Join(root, "blocker")
		assertNoError(t, os.WriteFile(conflict, nil, 0o600))
		path := filepath.Join(conflict, "manifest.json")
		if err := manifest.Write(path, manifest.New("run-1")); err == nil {
			t.Fatalf("Write should fail when MkdirAll cannot create the parent")
		}
	})

	t.Run("returns an error when WriteFile fails on a read-only directory", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		ro := filepath.Join(root, "ro")
		assertNoError(t, os.MkdirAll(ro, 0o750))
		assertNoError(t, os.Chmod(ro, 0o500))         //nolint:gosec // intentional read-only
		t.Cleanup(func() { _ = os.Chmod(ro, 0o750) }) //nolint:gosec // restore for TempDir cleanup
		if err := manifest.Write(filepath.Join(ro, "manifest.json"), manifest.New("run-1")); err == nil {
			t.Fatalf("Write should fail when WriteFile cannot create the file")
		}
	})

	t.Run("returns an error when Rename targets an existing directory", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		path := filepath.Join(root, "manifest.json")
		// Pre-create manifest.json as a directory.
		assertNoError(t, os.MkdirAll(path, 0o750))
		if err := manifest.Write(path, manifest.New("run-1")); err == nil {
			t.Fatalf("Write should fail when Rename collides with an existing directory")
		}
	})
}

func TestRead(t *testing.T) {
	t.Parallel()

	t.Run("round-trips a written manifest", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		path := filepath.Join(root, "manifest.json")
		original := manifest.New("run-1")
		original.Brand = "acmegen"
		original.Add(
			manifest.Output{
				Target:  targetAt("a", "b.go"),
				Plugins: []manifest.PluginAttribution{{Name: "p"}},
				Hash:    "sha256:x",
			},
		)
		assertNoError(t, manifest.Write(path, original))

		got, err := manifest.Read(path)
		assertNoError(t, err)
		if got.RunID != "run-1" {
			t.Fatalf("RunID = %q, want run-1", got.RunID)
		}
		if got.Brand != "acmegen" {
			t.Fatalf("Brand round-trip mismatch: got %q, want acmegen", got.Brand)
		}
		if len(got.Outputs) != 1 || got.Outputs[0].Target.Filename != "b.go" {
			t.Fatalf("Outputs round-trip mismatch: %+v", got.Outputs)
		}
	})

	t.Run("returns a wrapped filesystem error when the file is missing", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		_, err := manifest.Read(filepath.Join(root, "missing.json"))
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Read should propagate os.ErrNotExist; got %v", err)
		}
	})

	t.Run("returns a wrapped JSON error for malformed input", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		path := filepath.Join(root, "bad.json")
		assertNoError(t, os.WriteFile(path, []byte("not valid json"), 0o600))
		if _, err := manifest.Read(path); err == nil {
			t.Fatalf("Read should return an error for malformed JSON")
		}
	})

	t.Run("returns ErrUnsupportedVersion when version mismatches", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		path := filepath.Join(root, "future.json")
		assertNoError(
			t,
			os.WriteFile(path, []byte(`{"version":99,"run_id":"x","outputs":[]}`), 0o600),
		)
		_, err := manifest.Read(path)
		if !errors.Is(err, manifest.ErrUnsupportedVersion) {
			t.Fatalf("expected ErrUnsupportedVersion; got %v", err)
		}
	})
}
