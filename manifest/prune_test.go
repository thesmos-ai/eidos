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

		got := manifest.Prune(prev, current)
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
		got := manifest.Prune(prev, current)
		if len(got) != 3 || got[0].Filename != "first.go" || got[2].Filename != "third.go" {
			t.Fatalf("Prune should preserve prev order; got %+v", got)
		}
	})

	t.Run("returns nil when prev is nil", func(t *testing.T) {
		t.Parallel()
		current := manifest.New("current")
		current.Add(manifest.Output{Target: targetAt("a", "x.go")})
		if got := manifest.Prune(nil, current); got != nil {
			t.Fatalf("Prune with nil prev should be nil; got %+v", got)
		}
	})

	t.Run("returns nil when prev is empty", func(t *testing.T) {
		t.Parallel()
		if got := manifest.Prune(manifest.New("prev"), manifest.New("current")); got != nil {
			t.Fatalf("Prune with empty prev should be nil; got %+v", got)
		}
	})

	t.Run("treats nil current as empty (every prev target is stale)", func(t *testing.T) {
		t.Parallel()
		prev := manifest.New("prev")
		prev.Add(manifest.Output{Target: targetAt("a", "x.go")})
		got := manifest.Prune(prev, nil)
		if len(got) != 1 {
			t.Fatalf("Prune with nil current should treat all prev as stale; got %+v", got)
		}
	})
}
