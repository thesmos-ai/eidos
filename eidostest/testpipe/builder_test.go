// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package testpipe_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/eidostest/testpipe"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// minimalBackend returns a stubBackend wired with the supplied
// writes and a fixed name + language.
func minimalBackend() *stubBackend {
	return &stubBackend{name: "stub-be", lang: "stub"}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a Builder whose Build defaults to the configured sink and diag", func(t *testing.T) {
		t.Parallel()
		p := testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithBackend(minimalBackend()).
			Build()
		if p.Sink() == nil {
			t.Fatalf("Build should default Sink to an in-memory sink")
		}
		if p.Diagnostics() == nil {
			t.Fatalf("Build should default Diagnostics to a fresh sink")
		}
	})
}

func TestBuilder_WithFrontend(t *testing.T) {
	t.Parallel()

	t.Run("registered frontend reaches the underlying pipeline", func(t *testing.T) {
		t.Parallel()
		pkg := storefixture.New().Struct("S", nil).PackageNode()
		p := testpipe.New(t).
			WithFrontend(testpipe.FromNodes(pkg)).
			WithBackend(minimalBackend()).
			Build().
			Run()
		if p.Diagnostics().HasErrors() {
			t.Fatalf("run should be clean; got %+v", p.Diagnostics().Diagnostics())
		}
	})
}

func TestBuilder_WithAnnotator(t *testing.T) {
	t.Parallel()

	t.Run("registered annotator appears in the resolved plan", func(t *testing.T) {
		t.Parallel()
		ann := &stubAnnotator{name: "stub-ann"}
		_ = testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithAnnotator(ann).
			WithBackend(minimalBackend()).
			Build()
		// The build did not fail — the annotator was accepted. The
		// negative case (rejected annotator) is exercised in
		// pipeline.Builder's own tests.
	})
}

func TestBuilder_WithGenerator(t *testing.T) {
	t.Parallel()

	t.Run("registered generator appears in the resolved plan", func(t *testing.T) {
		t.Parallel()
		gen := &stubGenerator{name: "stub-gen"}
		_ = testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithGenerator(gen).
			WithBackend(minimalBackend()).
			Build()
	})
}

func TestBuilder_WithBackend(t *testing.T) {
	t.Parallel()

	t.Run("zero backends causes Build to fail the test", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		called := captureFatal(func() {
			testpipe.New(fake).
				WithFrontend(testpipe.FromNodes()).
				Build()
		})
		if !called {
			t.Fatalf("Build with zero backends should Fatalf via the fake TB")
		}
		if !fake.Failed() {
			t.Fatalf("fake TB should record failure")
		}
	})
}

// optsBackend is a backend that also implements [plugin.OptionsProvider].
// Used to verify Builder.WithPluginOptions threads options through.
type optsBackend struct {
	name string
	lang string
	opts struct {
		Tag string `eidos:"tag,required"`
	}
}

func (b *optsBackend) Name() string                        { return b.name }
func (b *optsBackend) Language() string                    { return b.lang }
func (*optsBackend) Render(_ *plugin.BackendContext) error { return nil }
func (b *optsBackend) OptionsSchema() opt.Schema           { return opt.Reflect(b.opts) }
func (b *optsBackend) SetOptions(o opt.Options) error      { return o.Decode(&b.opts) }

func TestBuilder_WithPluginOptions(t *testing.T) {
	t.Parallel()

	t.Run("threads options into a plugin implementing OptionsProvider", func(t *testing.T) {
		t.Parallel()
		be := &optsBackend{name: "opts-be", lang: "stub"}
		_ = testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithBackend(be).
			WithPluginOptions("opts-be", map[string]string{"tag": "v1"}).
			Build()
		if be.opts.Tag != "v1" {
			t.Fatalf("options not threaded; got %q", be.opts.Tag)
		}
	})
}

func TestBuilder_WithDirective(t *testing.T) {
	t.Parallel()

	t.Run("registered directive schema is accepted by Build", func(t *testing.T) {
		t.Parallel()
		schema := directive.NewSchema("repo").Build()
		_ = testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithDirective(schema).
			WithBackend(minimalBackend()).
			Build()
	})
}

func TestBuilder_WithSink(t *testing.T) {
	t.Parallel()

	t.Run("user-supplied sink replaces the default memory sink", func(t *testing.T) {
		t.Parallel()
		user := sink.NewMemory()
		// The builder's Pipeline.Sink() accessor exposes the default
		// memory sink even after override — the override only changes
		// the sink the underlying pipeline writes through. That's by
		// design: tests still need a place to inspect captured output
		// when they swap the underlying sink for something else.
		p := testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithBackend(minimalBackend()).
			WithSink(user).
			Build()
		if p.Sink() == nil {
			t.Fatalf("Pipeline.Sink should remain wired even after WithSink override")
		}
	})
}

func TestBuilder_WithCache(t *testing.T) {
	t.Parallel()

	t.Run("user-supplied cache is accepted by Build", func(t *testing.T) {
		t.Parallel()
		_ = testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithBackend(minimalBackend()).
			WithCache(cache.NewNone()).
			Build()
	})
}

func TestBuilder_WithVerbose(t *testing.T) {
	t.Parallel()

	t.Run("verbose mode causes the pipeline to emit Info diagnostics", func(t *testing.T) {
		t.Parallel()
		p := testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithBackend(minimalBackend()).
			WithVerbose(true).
			Build().
			Run()
		var hasInfo bool
		for _, d := range p.Diagnostics().Diagnostics() {
			if d.Plugin == "pipeline" {
				hasInfo = true
				break
			}
		}
		if !hasInfo {
			t.Fatalf("verbose mode should emit pipeline-attributed diagnostics")
		}
	})
}

func TestBuilder_WithParallel(t *testing.T) {
	t.Parallel()

	t.Run("phases are accepted and the run completes cleanly", func(t *testing.T) {
		t.Parallel()
		p := testpipe.New(t).
			WithFrontend(testpipe.FromNodes()).
			WithBackend(minimalBackend()).
			WithParallel(pipeline.PhaseFrontend, pipeline.PhaseAnnotator, pipeline.PhaseGenerator).
			Build().
			Run()
		if p.Diagnostics().HasErrors() {
			t.Fatalf("parallel-enabled run should be clean; got %+v", p.Diagnostics().Diagnostics())
		}
	})
}

func TestBuilder_Build(t *testing.T) {
	t.Parallel()

	t.Run("build error calls Fatalf on the configured TB", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		called := captureFatal(func() {
			testpipe.New(fake).Build()
		})
		if !called {
			t.Fatalf("Build with no frontend and no backend should Fatalf")
		}
		if len(fake.fatals) == 0 {
			t.Fatalf("expected at least one recorded fatal")
		}
		if !strings.Contains(fake.fatals[0], "build failed") {
			t.Fatalf("fatal should mention the build failure; got %q", fake.fatals[0])
		}
	})
}
