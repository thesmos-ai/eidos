// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"errors"
	"os"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
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

func TestPipeline_Run_ParallelFrontends(t *testing.T) {
	t.Parallel()

	t.Run("multiple frontends + patterns dispatch concurrently when PhaseFrontend is opted in", func(t *testing.T) {
		t.Parallel()
		fe1 := &recFE{name: "fe1"}
		fe2 := &recFE{name: "fe2"}
		p, err := pipeline.New().
			WithFrontend(fe1).
			WithFrontend(fe2).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			WithParallel(pipeline.PhaseFrontend).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "a", "b"))
		// 2 frontends × 2 patterns = 4 Load calls total across both.
		if len(fe1.loaded)+len(fe2.loaded) != 4 {
			t.Fatalf("expected 4 Load calls across both frontends; got fe1=%v fe2=%v", fe1.loaded, fe2.loaded)
		}
	})
}

func TestPipeline_Run_ParallelAnnotators(t *testing.T) {
	t.Parallel()

	t.Run("disjoint Provides + WithParallel runs the bucket concurrently", func(t *testing.T) {
		t.Parallel()
		ann1 := &stubAnnCapRec{
			name: "a", priority: priority.AnnotatorShape, provides: []string{"x"},
		}
		ann2 := &stubAnnCapRec{
			name: "b", priority: priority.AnnotatorShape, provides: []string{"y"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(ann1).
			WithAnnotator(ann2).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			WithParallel(pipeline.PhaseAnnotator).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if ann1.calls != 1 || ann2.calls != 1 {
			t.Fatalf("both annotators should run; got a=%d b=%d", ann1.calls, ann2.calls)
		}
	})

	t.Run("overlapping Provides forces sequential within the bucket", func(t *testing.T) {
		t.Parallel()
		ann1 := &stubAnnCapRec{
			name: "a", priority: priority.AnnotatorShape, provides: []string{"shared"},
		}
		// Different name → not a duplicate-provider Build error,
		// but Provides overlap → the parallel-safe check rejects.
		ann2 := &stubAnnCapRec{
			name: "b", priority: priority.AnnotatorRefinement, provides: []string{"shared"},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(ann1).
			WithAnnotator(ann2).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			WithParallel(pipeline.PhaseAnnotator).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if ann1.calls != 1 || ann2.calls != 1 {
			t.Fatalf("both annotators should still run; got a=%d b=%d", ann1.calls, ann2.calls)
		}
	})
}

func TestPipeline_Run_ParallelGenerators(t *testing.T) {
	t.Parallel()

	t.Run("all NodesOnly generators in a bucket run concurrently", func(t *testing.T) {
		t.Parallel()
		g1 := &stubGenNodesOnly{name: "g1", priority: priority.GeneratorFoundation, nodesOnly: true}
		g2 := &stubGenNodesOnly{name: "g2", priority: priority.GeneratorFoundation, nodesOnly: true}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(g1).
			WithGenerator(g2).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			WithParallel(pipeline.PhaseGenerator).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if g1.calls != 1 || g2.calls != 1 {
			t.Fatalf("both generators should run; got g1=%d g2=%d", g1.calls, g2.calls)
		}
	})

	t.Run("a non-NodesOnly generator forces sequential within the bucket", func(t *testing.T) {
		t.Parallel()
		g1 := &stubGenNodesOnly{name: "g1", priority: priority.GeneratorFoundation, nodesOnly: true}
		// g2 doesn't implement NodesOnly → bucket falls back to sequential.
		g2 := &recGen{name: "g2"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(g1).
			WithGenerator(g2).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			WithParallel(pipeline.PhaseGenerator).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if g1.calls != 1 || g2.calls != 1 {
			t.Fatalf("both generators should still run; got g1=%d g2=%d", g1.calls, g2.calls)
		}
	})
}

func TestPipeline_Run_RecordsCacheKeys(t *testing.T) {
	t.Parallel()

	t.Run("each plugin's ReadSet hash is written to the cache", func(t *testing.T) {
		t.Parallel()
		c := cache.NewDisk(t.TempDir())
		ann := &recAnn{name: "ann"}
		gen := &recGen{name: "gen"}
		be := &recBE{name: "be", lang: "stub"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(ann).
			WithGenerator(gen).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			WithCache(c).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		// Three plugins (ann, gen, be) each record a cache key; the
		// frontend phase does not record one because Frontend.Load
		// does not receive a Reader.
		if !c.Has("plugin:ann:reads:" + emptyReadSetHash()) {
			t.Fatalf("expected annotator cache key with empty-readset hash")
		}
		if !c.Has("plugin:gen:reads:" + emptyReadSetHash()) {
			t.Fatalf("expected generator cache key with empty-readset hash")
		}
		if !c.Has("plugin:be:reads:" + emptyReadSetHash()) {
			t.Fatalf("expected backend cache key with empty-readset hash")
		}
	})
}

func TestPipeline_Run_ParallelAnnotatorsWithPlainPlugin(t *testing.T) {
	t.Parallel()

	t.Run("a non-CapabilityProvider annotator still runs in parallel mode (no conflict)", func(t *testing.T) {
		t.Parallel()
		// stubAnn doesn't implement CapabilityProvider so its
		// Provides set is empty; disjoint check accepts.
		plain := &recAnn{name: "plain"}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(plain).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(sink.NewMemory()).
			WithParallel(pipeline.PhaseAnnotator).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if plain.calls != 1 {
			t.Fatalf("plain annotator should run; calls=%d", plain.calls)
		}
	})
}

func TestPipeline_Run_LibraryCommandLine(t *testing.T) {
	// Serial: mutates os.Args. No sibling test reads os.Args or
	// asserts on BackendContext.Command, so parallel siblings stay
	// safe across the mutation window.
	t.Run("commandLine returns the library marker when os.Args carries no positional arguments", func(t *testing.T) {
		orig := os.Args
		t.Cleanup(func() { os.Args = orig })
		os.Args = []string{"some-binary"}

		var captured string
		be := &recBE{
			name: "be",
			lang: "stub",
			render: func(ctx *plugin.BackendContext) {
				captured = ctx.Command
			},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if captured != "(library)" {
			t.Fatalf("BackendContext.Command = %q, want %q", captured, "(library)")
		}
	})
}
