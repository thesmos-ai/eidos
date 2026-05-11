// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

func TestPluginRoles_ComposeEndToEnd(t *testing.T) {
	t.Parallel()

	t.Run("frontend -> annotator -> generator -> backend pipes data through the shared store", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		r := store.NewReader(s)
		d := diag.New()
		out := newMemSink()

		fe := &stubFrontend{name: "stub-fe"}
		ann := &stubAnnotator{name: "stub-ann"}
		gen := &stubGenerator{name: "stub-gen"}
		be := &stubBackend{name: "stub-be"}

		assertNoError(t, fe.Load(&plugin.FrontendContext{Store: s, Diag: d, Pattern: "input"}))
		assertNoError(t, ann.Annotate(&plugin.AnnotatorContext{Store: s, Reader: r, Diag: d}))
		assertNoError(t, gen.Generate(&plugin.GeneratorContext{Store: s, Reader: r, Diag: d}))
		assertNoError(t, be.Render(&plugin.BackendContext{
			Store:   s,
			Reader:  r,
			Diag:    d,
			Sink:    out,
			Lang:    be.Language(),
			Plugins: []plugin.Plugin{fe, ann, gen, be},
			Ordered: []plugin.Plugin{fe, ann, gen, be},
		}))

		if got := out.files["out/x_gen.go"]; string(got) != "stub:User" {
			t.Fatalf("end-to-end output mismatch: %q", got)
		}
		// The reader's ReadSet should have captured iteration over
		// structs (annotator) and emit structs (backend).
		if !r.ReadSet().Has("node:structs") {
			t.Fatalf("annotator's structs iteration should be recorded")
		}
		if !r.ReadSet().Has("emit:structs") {
			t.Fatalf("backend's emit-structs iteration should be recorded")
		}
	})
}
