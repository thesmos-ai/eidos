// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/manifest"
)

func TestPrune(t *testing.T) {
	t.Parallel()

	scope := map[string]struct{}{"example.com/a": {}}

	t.Run("returns entries the current run did not re-emit", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("run-2")
		freshEntry := manifest.Output{
			Target:     targetAtPath("a", "fresh.go", "example.com/a"),
			PipelineID: "p",
		}
		staleEntry := manifest.Output{
			Target:     targetAtPath("a", "stale.go", "example.com/a"),
			PipelineID: "p",
		}
		prev.Add(freshEntry)
		prev.Add(staleEntry)
		// Current run emitted only the fresh target.
		emitted := map[emit.Target]struct{}{freshEntry.Target: {}}

		got := manifest.Prune(prev, emitted, scope, "p")
		if len(got) != 1 || got[0].Target.Filename != "stale.go" {
			t.Fatalf("Prune should return only the un-claimed entry; got %+v", got)
		}
	})

	t.Run("scope filter excludes out-of-scope entries", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("run-2")
		prev.Add(manifest.Output{
			Target:     targetAtPath("b", "stale.go", "example.com/b"),
			PipelineID: "p",
		})
		// Un-emitted, but its import path is outside the scope set —
		// current run did not load this package, so prune must not
		// consider it an orphan.
		emitted := map[emit.Target]struct{}{}
		if got := manifest.Prune(prev, emitted, scope, "p"); got != nil {
			t.Errorf("out-of-scope entry must not be returned; got %+v", got)
		}
	})

	t.Run("test-shifted import path matches non-test scope entry", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("run-2")
		prev.Add(manifest.Output{
			Target:     targetAtPath("a", "x_test.go", "example.com/a_test"),
			PipelineID: "p",
		})
		emitted := map[emit.Target]struct{}{}
		got := manifest.Prune(prev, emitted, scope, "p")
		if len(got) != 1 {
			t.Fatalf("`<pkg>_test` auto-shift entry must match non-test scope; got %+v", got)
		}
	})

	t.Run("PipelineID mismatch excludes the entry", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("run-2")
		prev.Add(manifest.Output{
			Target:     targetAtPath("a", "x_bench.go", "example.com/a"),
			PipelineID: "bench",
		})
		// Un-emitted, in scope, but owned by a different pipeline —
		// prune for "suite" must not touch it.
		emitted := map[emit.Target]struct{}{}
		if got := manifest.Prune(prev, emitted, scope, "suite"); got != nil {
			t.Errorf("other-pipeline entry must not be returned; got %+v", got)
		}
	})

	t.Run("empty pipelineID returns nil (refuses to scope without identity)", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("run-2")
		prev.Add(manifest.Output{
			Target: targetAtPath("a", "x.go", "example.com/a"),
		})
		emitted := map[emit.Target]struct{}{}
		if got := manifest.Prune(prev, emitted, scope, ""); got != nil {
			t.Errorf("empty pipelineID must not return candidates; got %+v", got)
		}
	})

	t.Run("nil prev / nil scope / nil emitted all return nil", func(t *testing.T) {
		t.Parallel()
		if got := manifest.Prune(nil, map[emit.Target]struct{}{}, scope, "p"); got != nil {
			t.Errorf("nil prev must return nil; got %+v", got)
		}
		if got := manifest.Prune(manifest.New("r"), map[emit.Target]struct{}{}, nil, "p"); got != nil {
			t.Errorf("nil scope must return nil; got %+v", got)
		}
		if got := manifest.Prune(manifest.New("r"), nil, scope, "p"); got != nil {
			t.Errorf("nil emitted must return nil; got %+v", got)
		}
	})

	t.Run("preserves manifest order in the returned slice", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("run-2")
		prev.Add(manifest.Output{
			Target:     targetAtPath("a", "first.go", "example.com/a"),
			PipelineID: "p",
		})
		prev.Add(manifest.Output{
			Target:     targetAtPath("a", "second.go", "example.com/a"),
			PipelineID: "p",
		})
		emitted := map[emit.Target]struct{}{}
		got := manifest.Prune(prev, emitted, scope, "p")
		if len(got) != 2 ||
			got[0].Target.Filename != "first.go" ||
			got[1].Target.Filename != "second.go" {

			t.Fatalf("Prune must preserve order; got %+v", got)
		}
	})
}
