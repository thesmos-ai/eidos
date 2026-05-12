// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"testing"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/sink"
)

func buildBasic(t *testing.T) *pipeline.Pipeline {
	t.Helper()
	p, err := pipeline.New().
		WithFrontend(&stubFE{name: "fe1"}).
		WithFrontend(&stubFE{name: "fe2"}).
		WithAnnotator(&stubAnn{name: "ann1"}).
		WithAnnotator(&stubAnn{name: "ann2"}).
		WithGenerator(&stubGen{name: "gen1"}).
		WithGenerator(&stubGen{name: "gen2"}).
		WithBackend(&stubBE{name: "be"}).
		WithSink(sink.NewMemory()).
		WithCache(cache.NewNone()).
		WithVerbose(true).
		Build()
	assertNoError(t, err)
	return p
}

func TestPipeline_Frontends(t *testing.T) {
	t.Parallel()

	t.Run("returns frontends in registration order", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		got := p.Frontends()
		if len(got) != 2 || got[0].Name() != "fe1" || got[1].Name() != "fe2" {
			t.Fatalf("Frontends order mismatch: %+v", got)
		}
	})

	t.Run("returns a defensive copy", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		first := p.Frontends()
		first[0] = nil
		if p.Frontends()[0] == nil {
			t.Fatalf("Frontends should return a defensive copy")
		}
	})
}

func TestPipeline_Annotators(t *testing.T) {
	t.Parallel()

	t.Run("returns annotators in registration order", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		got := p.Annotators()
		if len(got) != 2 || got[0].Name() != "ann1" || got[1].Name() != "ann2" {
			t.Fatalf("Annotators order mismatch: %+v", got)
		}
	})

	t.Run("returns a defensive copy", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		first := p.Annotators()
		first[0] = nil
		if p.Annotators()[0] == nil {
			t.Fatalf("Annotators should return a defensive copy")
		}
	})
}

func TestPipeline_Generators(t *testing.T) {
	t.Parallel()

	t.Run("returns generators in registration order", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		got := p.Generators()
		if len(got) != 2 || got[0].Name() != "gen1" || got[1].Name() != "gen2" {
			t.Fatalf("Generators order mismatch: %+v", got)
		}
	})

	t.Run("returns a defensive copy", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		first := p.Generators()
		first[0] = nil
		if p.Generators()[0] == nil {
			t.Fatalf("Generators should return a defensive copy")
		}
	})
}

func TestPipeline_Backend(t *testing.T) {
	t.Parallel()

	t.Run("returns the registered backend", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		if p.Backend() == nil || p.Backend().Name() != "be" {
			t.Fatalf("Backend mismatch: %+v", p.Backend())
		}
	})
}

func TestPipeline_Sink(t *testing.T) {
	t.Parallel()

	t.Run("returns the configured sink", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		if p.Sink() == nil {
			t.Fatalf("Sink should be non-nil after WithSink")
		}
	})

	t.Run("returns nil when no sink was configured", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		if p.Sink() != nil {
			t.Fatalf("Sink should be nil when WithSink is omitted")
		}
	})
}

func TestPipeline_Cache(t *testing.T) {
	t.Parallel()

	t.Run("returns the configured cache", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		if p.Cache() == nil {
			t.Fatalf("Cache should be non-nil after WithCache")
		}
	})
}

func TestPipeline_Diag(t *testing.T) {
	t.Parallel()

	t.Run("returns the configured diagnostic sink", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		if p.Diag() == nil {
			t.Fatalf("Diag should be non-nil")
		}
	})
}

func TestPipeline_Verbose(t *testing.T) {
	t.Parallel()

	t.Run("reports the configured verbose flag", func(t *testing.T) {
		t.Parallel()
		if !buildBasic(t).Verbose() {
			t.Fatalf("Verbose should reflect the configured value")
		}
	})

	t.Run("defaults to false", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		if p.Verbose() {
			t.Fatalf("Verbose should default to false")
		}
	})
}

func TestPipeline_Plan(t *testing.T) {
	t.Parallel()

	t.Run("returns the resolved plan after a successful Build", func(t *testing.T) {
		t.Parallel()
		p := buildBasic(t)
		plan := p.Plan()
		if plan == nil {
			t.Fatalf("Plan should be non-nil after a successful Build")
		}
		if len(plan.Frontends) != 2 || len(plan.Annotators) != 2 || len(plan.Generators) != 2 {
			t.Fatalf("plan slice lengths mismatch: %+v", plan)
		}
		if plan.Backend == nil || plan.Backend.Name() != "be" {
			t.Fatalf("plan backend mismatch")
		}
	})
}

func TestPipeline_DirectiveRegistry(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil registry even when no schemas are registered", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		if p.DirectiveRegistry() == nil {
			t.Fatalf("DirectiveRegistry should be non-nil after Build")
		}
	})
}

// TestPipeline_Store covers the post-run store accessor. Before Run
// fires, Store returns nil — the cache is populated on Run entry, so
// tests inspecting plan/registry/backend state pre-run see the nil
// signal that no store yet exists. After Run, Store returns the same
// instance the phases populated, so post-run inspection of node /
// emit content goes through one canonical accessor.
func TestPipeline_Store(t *testing.T) {
	t.Parallel()

	t.Run("returns nil before Run is invoked", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		if p.Store() != nil {
			t.Fatalf("Store should be nil before Run; got %v", p.Store())
		}
	})

	t.Run("returns the populated store after Run completes", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		if p.Store() == nil {
			t.Fatalf("Store should be non-nil after Run")
		}
	})

	t.Run("re-running replaces the previously cached store", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		first := p.Store()
		assertNoError(t, p.Run(t.Context()))
		second := p.Store()
		if first == second {
			t.Fatalf("re-running should replace the cached store; got identical pointers")
		}
	})
}
