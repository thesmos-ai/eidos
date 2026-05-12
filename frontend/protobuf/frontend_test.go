// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/frontend/protobuf"
	"go.thesmos.sh/eidos/plugin"
)

// TestPluginShape pins the protobuf frontend's public-contract
// surface so a rename / drop accident surfaces at PR review time.
func TestPluginShape(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented identifier", func(t *testing.T) {
		t.Parallel()
		if got := protobuf.New().Name(); got != protobuf.FrontendName {
			t.Fatalf("Name = %q, want %q", got, protobuf.FrontendName)
		}
	})

	t.Run("FrontendName is the literal string \"protobuf\"", func(t *testing.T) {
		t.Parallel()
		if protobuf.FrontendName != "protobuf" {
			t.Fatalf("FrontendName = %q, want %q", protobuf.FrontendName, "protobuf")
		}
	})

	t.Run("Frontend implements plugin.Frontend", func(t *testing.T) {
		t.Parallel()
		var _ plugin.Frontend = protobuf.New()
	})

	t.Run("Frontend implements plugin.Versioned", func(t *testing.T) {
		t.Parallel()
		var _ plugin.Versioned = protobuf.New()
	})

	t.Run("Frontend implements plugin.OptionsProvider", func(t *testing.T) {
		t.Parallel()
		var _ plugin.OptionsProvider = protobuf.New()
	})

	t.Run("Version is non-empty so it contributes to the cache key", func(t *testing.T) {
		t.Parallel()
		if protobuf.New().Version() == "" {
			t.Fatalf("Version() = empty; per the spec the frontend must declare a version")
		}
	})
}

// TestOptionsSchema pins the documented frontend options surface.
// Each option's default and accepted value forms must round-trip
// through the reflected schema so misconfigured pipelines surface
// positioned diagnostics at Build time.
func TestOptionsSchema(t *testing.T) {
	t.Parallel()

	t.Run("OptionsSchema returns a schema with the documented keys", func(t *testing.T) {
		t.Parallel()
		schema := protobuf.New().OptionsSchema()
		wantKeys := []string{"dir", "import_paths", "include_well_known"}
		for _, key := range wantKeys {
			if !schema.HasField(key) {
				t.Fatalf("schema missing option %q", key)
			}
		}
	})

	t.Run("default option values match the documented defaults", func(t *testing.T) {
		t.Parallel()
		f := protobuf.New()
		assertNoError(t, f.SetOptions(opt.New(f.OptionsSchema(), nil)))
		if got := f.Dir(); got != "" {
			t.Fatalf("dir default = %q, want empty", got)
		}
		if got := f.ImportPaths(); len(got) != 0 {
			t.Fatalf("import_paths default = %v, want empty slice", got)
		}
		if got := f.IncludeWellKnown(); !got {
			t.Fatalf("include_well_known default = false, want true")
		}
	})

	t.Run("import_paths parses a comma-separated list", func(t *testing.T) {
		t.Parallel()
		f := protobuf.New()
		assertNoError(t, f.SetOptions(opt.New(f.OptionsSchema(), map[string]string{
			"import_paths": "/a,/b,/c",
		})))
		got := f.ImportPaths()
		want := []string{"/a", "/b", "/c"}
		if len(got) != len(want) {
			t.Fatalf("import_paths len = %d, want %d (%v)", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("import_paths[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("include_well_known accepts true / false", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			raw  string
			want bool
		}{{"true", true}, {"false", false}} {
			f := protobuf.New()
			assertNoError(t, f.SetOptions(opt.New(f.OptionsSchema(), map[string]string{
				"include_well_known": tc.raw,
			})))
			if got := f.IncludeWellKnown(); got != tc.want {
				t.Fatalf("include_well_known(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		}
	})
}

// assertNoError fails the test if err is non-nil. Mirrors the
// helper used across the rest of the package's tests; lives here
// in the per-file test until helpers_test.go lands with the
// shared fixtures.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
