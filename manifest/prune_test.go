// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"testing"

	"go.thesmos.sh/eidos/manifest"
)

func TestPrune(t *testing.T) {
	t.Parallel()

	t.Run("returns targets in prev but not in current", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("prev")
		prev.Add(manifest.Output{Target: targetAt("a", "kept.go")})
		prev.Add(manifest.Output{Target: targetAt("a", "stale.go")})
		current := manifest.New("current")
		current.Add(manifest.Output{Target: targetAt("a", "kept.go")})

		got := manifest.Prune(prev, current, "")
		if len(got) != 1 || got[0].Filename != "stale.go" {
			t.Fatalf("Prune mismatch: %+v", got)
		}
	})

	t.Run("preserves prev's order in the returned slice", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("prev")
		prev.Add(manifest.Output{Target: targetAt("a", "first.go")})
		prev.Add(manifest.Output{Target: targetAt("a", "second.go")})
		prev.Add(manifest.Output{Target: targetAt("a", "third.go")})
		current := manifest.New("current")
		// current claims none of them — every prev target is stale.
		got := manifest.Prune(prev, current, "")
		if len(got) != 3 || got[0].Filename != "first.go" || got[2].Filename != "third.go" {
			t.Fatalf("Prune should preserve prev order; got %+v", got)
		}
	})

	t.Run("returns nil when prev is nil", func(t *testing.T) {
		t.Parallel()
		current := manifest.New("current")
		current.Add(manifest.Output{Target: targetAt("a", "x.go")})
		if got := manifest.Prune(nil, current, ""); got != nil {
			t.Fatalf("Prune with nil prev should be nil; got %+v", got)
		}
	})

	t.Run("returns nil when prev is empty", func(t *testing.T) {
		t.Parallel()
		if got := manifest.Prune(manifest.New("prev"), manifest.New("current"), ""); got != nil {
			t.Fatalf("Prune with empty prev should be nil; got %+v", got)
		}
	})

	t.Run("treats nil current as empty (every prev target is stale)", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("prev")
		prev.Add(manifest.Output{Target: targetAt("a", "x.go")})
		got := manifest.Prune(prev, nil, "")
		if len(got) != 1 {
			t.Fatalf("Prune with nil current should treat all prev as stale; got %+v", got)
		}
	})

	t.Run("non-empty pipelineID scopes to that pipeline's entries", func(t *testing.T) {
		t.Parallel()
		// prev holds one entry from "bench" pipeline and one from
		// "suite" pipeline; current has neither (both pipelines have
		// fully unclaimed their files). Prune for "bench" must
		// return ONLY the bench entry — the suite entry is owned by
		// a different pipeline and is off-limits.
		prev := manifest.New("prev")
		prev.Add(manifest.Output{
			Target: targetAt("a", "x_bench_test.go"), PipelineID: "bench",
		})
		prev.Add(manifest.Output{
			Target: targetAt("a", "x_suite_test.go"), PipelineID: "suite",
		})
		got := manifest.Prune(prev, manifest.New("current"), "bench")
		if len(got) != 1 || got[0].Filename != "x_bench_test.go" {
			t.Errorf("scoped Prune must only return bench entries; got %+v", got)
		}
	})
}
