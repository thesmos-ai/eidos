// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protopipe_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/protopipe"
)

// TestRun_FrontendOnly covers the frontend-only entry point: a
// proto source fixture drives the protobuf frontend through the
// full pipeline (with a no-op backend) without emitting
// diagnostics. Asserts that the harness's defaults — Pattern,
// Sink, Cache, no-op Backend — produce a clean run.
func TestRun_FrontendOnly(t *testing.T) {
	t.Parallel()

	t.Run("simple fixture runs with no diagnostics", func(t *testing.T) {
		t.Parallel()
		root := protopipe.FixtureRoot(t, "simple")
		result := protopipe.Run(t, protopipe.RunOptions{SourceDir: root})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		if result.RunErr != nil {
			t.Fatalf("Run returned error: %v", result.RunErr)
		}
		if result.Store == nil {
			t.Fatalf("Result.Store is nil; the pipeline should always allocate a store")
		}
	})
}

// TestAssertDeterministic covers the determinism harness helper:
// two consecutive runs against the same fixture must serialize
// to the same node-side bytes. The helper works against any
// store contents — empty or populated.
func TestAssertDeterministic(t *testing.T) {
	t.Parallel()

	t.Run("two consecutive runs produce identical node graphs", func(t *testing.T) {
		t.Parallel()
		root := protopipe.FixtureRoot(t, "simple")
		protopipe.AssertDeterministic(t, protopipe.RunOptions{SourceDir: root})
	})
}

// TestLoadDirect_FrontendOnly covers the lower-level entry point
// callers use when they need the frontend's diagnostics without
// the pipeline-build invariants.
func TestLoadDirect_FrontendOnly(t *testing.T) {
	t.Parallel()

	t.Run("simple fixture loads without diagnostics", func(t *testing.T) {
		t.Parallel()
		root := protopipe.FixtureRoot(t, "simple")
		result := protopipe.LoadDirect(t, protopipe.RunOptions{SourceDir: root})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		if result.Store == nil {
			t.Fatalf("Result.Store is nil; LoadDirect should allocate a store")
		}
	})
}
