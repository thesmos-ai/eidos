// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/plugin"
)

func TestAnnotator_Contract(t *testing.T) {
	t.Parallel()

	t.Run("Annotate stamps metadata on every reachable struct", func(t *testing.T) {
		t.Parallel()
		s, r := makeStubReader(t)
		var ann plugin.Annotator = &stubAnnotator{name: "stub-ann"}
		assertNoError(t, ann.Annotate(&plugin.AnnotatorContext{
			Store:  s,
			Reader: r,
			Diag:   diag.New(),
		}))
		got := s.Nodes().ByMetaKey().Get(stubKey.Name())
		if len(got) != 1 {
			t.Fatalf("ByMetaKey = %d, want 1 (one struct annotated)", len(got))
		}
	})
}
