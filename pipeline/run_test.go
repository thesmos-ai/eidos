// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"errors"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

func TestPipeline_Run_NoSink(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrNoSink when no sink was configured", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		if got := p.Run(t.Context()); !errors.Is(got, pipeline.ErrNoSink) {
			t.Fatalf("Run without sink should return ErrNoSink; got %v", got)
		}
	})
}

func TestPipeline_Run_FrontendPatterns(t *testing.T) {
	t.Parallel()

	t.Run("each frontend receives every supplied pattern in order", func(t *testing.T) {
		t.Parallel()
		fe := &recFE{name: "fe"}
		p, err := pipeline.New().
			WithFrontend(fe).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "pkg/a", "pkg/b"))
		if !slices.Equal(fe.loaded, []string{"pkg/a", "pkg/b"}) {
			t.Fatalf("frontend should see every pattern in order; got %v", fe.loaded)
		}
	})

	t.Run("multiple frontends each see every pattern", func(t *testing.T) {
		t.Parallel()
		fe1 := &recFE{name: "fe1"}
		fe2 := &recFE{name: "fe2"}
		p, err := pipeline.New().
			WithFrontend(fe1).
			WithFrontend(fe2).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if !slices.Equal(fe1.loaded, []string{"x"}) || !slices.Equal(fe2.loaded, []string{"x"}) {
			t.Fatalf("each frontend should receive every pattern; got fe1=%v fe2=%v", fe1.loaded, fe2.loaded)
		}
	})

	t.Run("a frontend Load error becomes an Error diagnostic and Run returns ErrRunHadErrors", func(t *testing.T) {
		t.Parallel()
		fe := &recFE{name: "fe", err: errors.New("load failed")}
		p, err := pipeline.New().
			WithFrontend(fe).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		got := p.Run(t.Context(), "pkg")
		if !errors.Is(got, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run should return ErrRunHadErrors after a frontend failure; got %v", got)
		}
		if p.Diag().Count(diag.Error) == 0 {
			t.Fatalf("frontend error should surface as a diag.Error")
		}
	})
}

func TestPipeline_Run_AnnotatorPhase(t *testing.T) {
	t.Parallel()

	t.Run("every annotator runs in plan order against the shared store", func(t *testing.T) {
		t.Parallel()
		var order []string
		mark := func(name string) func(*plugin.AnnotatorContext) {
			return func(*plugin.AnnotatorContext) { order = append(order, name) }
		}
		ann1 := &recAnn{name: "ann1", annotate: mark("ann1")}
		ann2 := &recAnn{name: "ann2", annotate: mark("ann2")}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(ann1).
			WithAnnotator(ann2).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if !slices.Equal(order, []string{"ann1", "ann2"}) {
			t.Fatalf("annotator plan order mismatch: %v", order)
		}
	})

	t.Run("an annotator error becomes an Error diagnostic and execution continues", func(t *testing.T) {
		t.Parallel()
		ann1 := &recAnn{name: "ann1", err: errors.New("first failed")}
		ann2 := &recAnn{name: "ann2"}
		be := &recBE{name: "be", lang: "stub"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(ann1).
			WithAnnotator(ann2).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		got := p.Run(t.Context())
		if !errors.Is(got, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run should return ErrRunHadErrors; got %v", got)
		}
		if ann2.calls != 1 {
			t.Fatalf("ann2 should still run after ann1 fails; calls=%d", ann2.calls)
		}
		if be.calls != 1 {
			t.Fatalf("backend should still run after ann1 fails; calls=%d", be.calls)
		}
	})
}

func TestPipeline_Run_GeneratorPhase(t *testing.T) {
	t.Parallel()

	t.Run("every generator runs in plan order", func(t *testing.T) {
		t.Parallel()
		var order []string
		mark := func(name string) func(*plugin.GeneratorContext) {
			return func(*plugin.GeneratorContext) { order = append(order, name) }
		}
		g1 := &recGen{name: "g1", generate: mark("g1")}
		g2 := &recGen{name: "g2", generate: mark("g2")}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(g1).
			WithGenerator(g2).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if !slices.Equal(order, []string{"g1", "g2"}) {
			t.Fatalf("generator plan order mismatch: %v", order)
		}
	})

	t.Run("a generator error becomes an Error diagnostic and execution continues", func(t *testing.T) {
		t.Parallel()
		g1 := &recGen{name: "g1", err: errors.New("first failed")}
		g2 := &recGen{name: "g2"}
		be := &recBE{name: "be", lang: "stub"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(g1).
			WithGenerator(g2).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		_ = p.Run(t.Context())
		if g2.calls != 1 {
			t.Fatalf("g2 should still run after g1 fails; calls=%d", g2.calls)
		}
		if be.calls != 1 {
			t.Fatalf("backend should still run after g1 fails; calls=%d", be.calls)
		}
	})
}

func TestPipeline_Run_BackendPhase(t *testing.T) {
	t.Parallel()

	t.Run("backend receives the configured sink and registered/ordered plugin lists", func(t *testing.T) {
		t.Parallel()
		var seenCtx *plugin.BackendContext
		be := &recBE{
			name: "be", lang: "stub",
			render: func(ctx *plugin.BackendContext) { seenCtx = ctx },
		}
		ann := &stubAnn{name: "ann"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(ann).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if seenCtx == nil {
			t.Fatalf("backend should have been called")
		}
		if seenCtx.Sink == nil {
			t.Fatalf("BackendContext.Sink should be the configured sink")
		}
		if seenCtx.Lang != "stub" {
			t.Fatalf("BackendContext.Lang = %q, want stub", seenCtx.Lang)
		}
		if len(seenCtx.Plugins) == 0 || len(seenCtx.Ordered) == 0 {
			t.Fatalf("Plugins / Ordered should be populated; got %+v", seenCtx)
		}
	})

	t.Run("a backend Render error becomes an Error diagnostic", func(t *testing.T) {
		t.Parallel()
		be := &recBE{name: "be", lang: "stub", err: errors.New("render failed")}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		got := p.Run(t.Context())
		if !errors.Is(got, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run should return ErrRunHadErrors; got %v", got)
		}
	})
}

func TestPipeline_Run_EndToEnd(t *testing.T) {
	t.Parallel()

	t.Run("frontend -> annotator -> generator -> backend pipes data through the store and sink", func(t *testing.T) {
		t.Parallel()
		mem := sink.NewMemory()
		// Frontend populates one struct on every Load call.
		fe := &recFE{
			name: "fe",
			loadFn: func(s *store.Store) {
				_ = s.Nodes().AddPackage(&node.Package{
					Name: "x", Path: "x",
					Structs: []*node.Struct{{Name: "User", Package: "x"}},
				})
			},
		}
		// Generator emits one struct.
		gen := &recGen{
			name: "gen",
			generate: func(ctx *plugin.GeneratorContext) {
				_ = ctx.Store.Emit().AddPackage(&emit.Package{
					Name: "x", Path: "x", Dir: "out",
					Structs: []*emit.Struct{{
						Name: "User", Package: "x",
						Target: emit.Target{Dir: "out", Filename: "user_gen.go", Package: "x"},
					}},
				})
			},
		}
		// Backend writes one byte payload per emit struct.
		be := &recBE{
			name: "be", lang: "stub",
			render: func(ctx *plugin.BackendContext) {
				ctx.Reader.EmitStructs().Each(func(s *emit.Struct) {
					_ = ctx.Sink.Write(s.Target, []byte("rendered:"+s.Name))
				})
			},
		}
		p, err := pipeline.New().
			WithFrontend(fe).
			WithGenerator(gen).
			WithBackend(be).
			WithSink(mem).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		got, ok := mem.Get(emit.Target{Dir: "out", Filename: "user_gen.go", Package: "x"})
		if !ok || string(got) != "rendered:User" {
			t.Fatalf("end-to-end output mismatch: %q ok=%v", got, ok)
		}
	})
}

func TestPipeline_DryRun(t *testing.T) {
	t.Parallel()

	t.Run("returns the resolved Plan without executing any phase", func(t *testing.T) {
		t.Parallel()
		fe := &recFE{name: "fe"}
		ann := &recAnn{name: "ann"}
		gen := &recGen{name: "gen"}
		be := &recBE{name: "be", lang: "stub"}
		p, err := pipeline.New().
			WithFrontend(fe).
			WithAnnotator(ann).
			WithGenerator(gen).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		got := p.DryRun(t.Context())
		if got == nil || got.Backend == nil {
			t.Fatalf("DryRun should return the resolved plan; got %+v", got)
		}
		if len(fe.loaded) != 0 || ann.calls != 0 || gen.calls != 0 || be.calls != 0 {
			t.Fatalf("DryRun must not execute any phase")
		}
	})
}

func TestPipeline_Run_PanicRecovery(t *testing.T) {
	t.Parallel()

	t.Run("a panicking frontend becomes an Error diagnostic and the run continues", func(t *testing.T) {
		t.Parallel()
		be := &recBE{name: "be", lang: "stub"}
		p, err := pipeline.New().
			WithFrontend(&panickyFE{name: "fe", msg: "fe boom"}).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		got := p.Run(t.Context(), "pkg")
		if !errors.Is(got, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run should return ErrRunHadErrors after a panic; got %v", got)
		}
		if be.calls != 1 {
			t.Fatalf("backend should still run after a frontend panic; calls=%d", be.calls)
		}
		if !hasPanicMessage(p.Diag(), "fe boom") {
			t.Fatalf("panic message should be captured in diagnostics")
		}
	})

	t.Run("a panicking annotator becomes an Error diagnostic and subsequent phases run", func(t *testing.T) {
		t.Parallel()
		be := &recBE{name: "be", lang: "stub"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(&panickyAnn{name: "ann", msg: "ann boom"}).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		_ = p.Run(t.Context())
		if be.calls != 1 {
			t.Fatalf("backend should still run after an annotator panic")
		}
		if !hasPanicMessage(p.Diag(), "ann boom") {
			t.Fatalf("annotator panic message not captured")
		}
	})

	t.Run("a panicking generator becomes an Error diagnostic and the backend still runs", func(t *testing.T) {
		t.Parallel()
		be := &recBE{name: "be", lang: "stub"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(&panickyGen{name: "gen", msg: "gen boom"}).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		_ = p.Run(t.Context())
		if be.calls != 1 {
			t.Fatalf("backend should still run after a generator panic")
		}
		if !hasPanicMessage(p.Diag(), "gen boom") {
			t.Fatalf("generator panic message not captured")
		}
	})

	t.Run("a panicking backend becomes an Error diagnostic", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&panickyBE{name: "be", lang: "stub", msg: "be boom"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		got := p.Run(t.Context())
		if !errors.Is(got, pipeline.ErrRunHadErrors) {
			t.Fatalf("Run should return ErrRunHadErrors after a backend panic; got %v", got)
		}
		if !hasPanicMessage(p.Diag(), "be boom") {
			t.Fatalf("backend panic message not captured")
		}
	})
}

func TestPipeline_Run_FreezeContract(t *testing.T) {
	t.Parallel()

	t.Run("an annotator that mutates the frozen node view produces an Internal diagnostic", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(&frozenAddAnn{name: "bad-ann"}).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		_ = p.Run(t.Context())
		if len(internalDiagsFor(p.Diag())) == 0 {
			t.Fatalf("frozen-store violation should surface as an Internal diagnostic")
		}
	})

	t.Run("a backend that mutates the frozen emit view produces an Internal diagnostic", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&frozenAddBE{name: "bad-be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		_ = p.Run(t.Context())
		if len(internalDiagsFor(p.Diag())) == 0 {
			t.Fatalf("frozen-emit violation should surface as an Internal diagnostic")
		}
	})
}

func TestPipeline_Run_VerbosePhaseLogs(t *testing.T) {
	t.Parallel()

	t.Run("verbose mode emits one phase-boundary Info per phase plus a run summary", func(t *testing.T) {
		t.Parallel()
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			WithVerbose(true).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if got := phaseLogs(d); len(got) != 5 {
			t.Fatalf("expected 5 phase logs (FE/Ann/Override/Gen/BE); got %d: %v", len(got), got)
		}
		// Plus the run-summary Info; total Info >= 6.
		if d.Count(diag.Info) < 6 {
			t.Fatalf("expected at least 6 Info diagnostics (5 phases + summary); got %d", d.Count(diag.Info))
		}
	})

	t.Run("non-verbose runs emit no Info diagnostics", func(t *testing.T) {
		t.Parallel()
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if d.Count(diag.Info) != 0 {
			t.Fatalf("non-verbose run should emit no Info diagnostics; got %d", d.Count(diag.Info))
		}
	})
}
