// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package demopipe_test

import (
	"os"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/demopipe"
	"go.thesmos.sh/eidos/sink"
)

// TestFixtureRoot pins that the harness resolves the demoproject
// fixture path correctly and that the resolved directory carries
// the expected layout markers.
func TestFixtureRoot(t *testing.T) {
	t.Parallel()

	t.Run("resolves to an existing directory carrying go.mod", func(t *testing.T) {
		t.Parallel()
		root := demopipe.FixtureRoot(t)
		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("FixtureRoot %q stat: %v", root, err)
		}
		if !info.IsDir() {
			t.Fatalf("FixtureRoot %q is not a directory", root)
		}
		if _, err := os.Stat(root + "/go.mod"); err != nil {
			t.Fatalf("FixtureRoot %q missing go.mod: %v", root, err)
		}
		if _, err := os.Stat(root + "/blog"); err != nil {
			t.Fatalf("FixtureRoot %q missing blog/: %v", root, err)
		}
		if _, err := os.Stat(root + "/extras"); err != nil {
			t.Fatalf("FixtureRoot %q missing extras/: %v", root, err)
		}
	})
}

// TestBaselineEmptyPipeline covers the empty-plugin-set baseline:
// the Go frontend parses the fixture, no annotator or generator
// transforms anything, and the backend renders nothing because the
// emit tree stays empty. Verifies the pipeline phases run end-to-
// end without errors and the empty-target filter does its job.
func TestBaselineEmptyPipeline(t *testing.T) {
	t.Parallel()

	t.Run("frontend-only pipeline produces zero sink writes and zero diagnostics", func(t *testing.T) {
		t.Parallel()
		result := demopipe.Run(t, demopipe.RunOptions{
			Backend: backend_golang.New(),
		})
		if result.RunErr != nil {
			t.Fatalf("baseline Run: %v", result.RunErr)
		}
		if result.Diag.HasErrors() {
			t.Fatalf("baseline run produced error diagnostics: %+v", result.Diag.Diagnostics())
		}
		mem, ok := result.Sink.(*sink.Memory)
		if !ok {
			t.Fatalf("expected default *sink.Memory; got %T", result.Sink)
		}
		if mem.Len() != 0 {
			t.Fatalf("baseline run produced %d sink writes; want 0 (files=%v)", mem.Len(), mem.Files())
		}
	})

	t.Run("frontend populates the node store with every fixture declaration", func(t *testing.T) {
		t.Parallel()
		result := demopipe.Run(t, demopipe.RunOptions{
			Backend: backend_golang.New(),
		})
		if result.Store == nil {
			t.Fatalf("baseline Run did not populate Store")
		}
		nodes := result.Store.Nodes()
		// Spot-check every fixture entity reaches the node bucket.
		for _, want := range []string{
			"example.com/demoproject/blog.Article",
			"example.com/demoproject/blog.User",
			"example.com/demoproject/blog.Comment",
			"example.com/demoproject/blog.LineWriter",
			"example.com/demoproject/blog.Box",
			"example.com/demoproject/blog.Score",
		} {
			if _, ok := nodes.Structs().ByQName(want); !ok {
				t.Errorf("node store missing struct %q", want)
			}
		}
		if _, ok := nodes.Interfaces().ByQName("example.com/demoproject/blog.Searcher"); !ok {
			t.Errorf("node store missing interface example.com/demoproject/blog.Searcher")
		}
		if _, ok := nodes.Enums().ByQName("example.com/demoproject/blog.Status"); !ok {
			t.Errorf("node store missing enum example.com/demoproject/blog.Status")
		}
	})
}
