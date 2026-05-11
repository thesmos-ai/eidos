// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/plugin"
)

func TestGenerator_Contract(t *testing.T) {
	t.Parallel()

	t.Run("Generate produces emit entities mirroring annotated source", func(t *testing.T) {
		t.Parallel()
		s, r := makeStubReader(t)
		// Annotate first so the generator has something to mirror.
		ann := &stubAnnotator{name: "stub-ann"}
		assertNoError(t, ann.Annotate(&plugin.AnnotatorContext{Store: s, Reader: r, Diag: diag.New()}))

		var gen plugin.Generator = &stubGenerator{name: "stub-gen"}
		assertNoError(t, gen.Generate(&plugin.GeneratorContext{
			Store:  s,
			Reader: r,
			Diag:   diag.New(),
		}))
		if s.Emit().Structs().Len() == 0 {
			t.Fatalf("Generate should have populated the emit store")
		}
	})
}
