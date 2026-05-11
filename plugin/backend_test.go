// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/plugin"
)

func TestBackend_Contract(t *testing.T) {
	t.Parallel()

	t.Run("Language returns the configured target identifier", func(t *testing.T) {
		t.Parallel()
		var be plugin.Backend = &stubBackend{name: "stub"}
		if be.Language() != "stub" {
			t.Fatalf("Language = %q, want stub", be.Language())
		}
	})

	t.Run("Render writes one payload per emit struct through the sink", func(t *testing.T) {
		t.Parallel()
		s, r := makeStubReader(t)
		ann := &stubAnnotator{name: "stub-ann"}
		gen := &stubGenerator{name: "stub-gen"}
		assertNoError(t, ann.Annotate(&plugin.AnnotatorContext{Store: s, Reader: r, Diag: diag.New()}))
		assertNoError(t, gen.Generate(&plugin.GeneratorContext{Store: s, Reader: r, Diag: diag.New()}))

		out := newMemSink()
		var be plugin.Backend = &stubBackend{name: "stub-be"}
		assertNoError(t, be.Render(&plugin.BackendContext{
			Store:  s,
			Reader: r,
			Diag:   diag.New(),
			Sink:   out,
			Lang:   be.Language(),
		}))
		if got := out.files["out/x_gen.go"]; string(got) != "stub:User" {
			t.Fatalf("sink content mismatch: %q", got)
		}
	})

	t.Run("Render propagates sink errors to the caller", func(t *testing.T) {
		t.Parallel()
		s, r := makeStubReader(t)
		ann := &stubAnnotator{name: "stub-ann"}
		gen := &stubGenerator{name: "stub-gen"}
		assertNoError(t, ann.Annotate(&plugin.AnnotatorContext{Store: s, Reader: r, Diag: diag.New()}))
		assertNoError(t, gen.Generate(&plugin.GeneratorContext{Store: s, Reader: r, Diag: diag.New()}))

		be := &stubBackend{name: "stub-be"}
		err := be.Render(&plugin.BackendContext{
			Store:  s,
			Reader: r,
			Diag:   diag.New(),
			Sink:   &failingSink{err: errSinkBoom},
			Lang:   be.Language(),
		})
		if !errors.Is(err, errSinkBoom) {
			t.Fatalf("Render should propagate sink error; got %v", err)
		}
	})

	t.Run("BackendContext carries Plugins and Ordered for template merging", func(t *testing.T) {
		t.Parallel()
		ann := &stubAnnotator{name: "stub-ann"}
		ctx := &plugin.BackendContext{
			Plugins: []plugin.Plugin{ann},
			Ordered: []plugin.Plugin{ann},
		}
		if len(ctx.Plugins) != 1 || ctx.Plugins[0].Name() != "stub-ann" {
			t.Fatalf("Plugins not threaded correctly")
		}
		if len(ctx.Ordered) != 1 || ctx.Ordered[0].Name() != "stub-ann" {
			t.Fatalf("Ordered not threaded correctly")
		}
	})
}
