// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/sink"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil Builder", func(t *testing.T) {
		t.Parallel()
		if pipeline.New() == nil {
			t.Fatalf("New should return a non-nil Builder")
		}
	})
}

func TestBuilder_With(t *testing.T) {
	t.Parallel()

	t.Run("With* methods return the receiver for chaining", func(t *testing.T) {
		t.Parallel()
		b := pipeline.New()
		out := b.WithFrontend(&stubFE{name: "fe"}).
			WithAnnotator(&stubAnn{name: "ann"}).
			WithGenerator(&stubGen{name: "gen"}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithCache(cache.NewNone()).
			WithDiag(diag.New()).
			WithVerbose(true).
			WithPluginOptions("p", map[string]string{"k": "v"})
		if out != b {
			t.Fatalf("With* should return the receiver")
		}
	})
}

func TestBuilder_Build(t *testing.T) {
	t.Parallel()

	t.Run("succeeds with one frontend, one backend, and no options", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		if p == nil {
			t.Fatalf("Build should return a non-nil Pipeline on success")
		}
	})

	t.Run("populates default cache and diag when not configured", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
		if p.Cache() == nil {
			t.Fatalf("Build should default Cache when not configured")
		}
		if p.Diag() == nil {
			t.Fatalf("Build should default Diag when not configured")
		}
	})

	t.Run("rejects duplicate plugin names with ErrDuplicatePlugin", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "shared"}).
			WithAnnotator(&stubAnn{name: "shared"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		if !errors.Is(err, pipeline.ErrDuplicatePlugin) {
			t.Fatalf("Build should return ErrDuplicatePlugin; got %v", err)
		}
	})

	t.Run("rejects zero frontends with ErrNoFrontend", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().WithBackend(&stubBE{name: "be"}).Build()
		if !errors.Is(err, pipeline.ErrNoFrontend) {
			t.Fatalf("Build should return ErrNoFrontend; got %v", err)
		}
	})

	t.Run("rejects zero backends with ErrNoBackend", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().WithFrontend(&stubFE{name: "fe"}).Build()
		if !errors.Is(err, pipeline.ErrNoBackend) {
			t.Fatalf("Build should return ErrNoBackend; got %v", err)
		}
	})

	t.Run("rejects multiple backends with ErrMultipleBackends", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be1"}).
			WithBackend(&stubBE{name: "be2"}).
			Build()
		if !errors.Is(err, pipeline.ErrMultipleBackends) {
			t.Fatalf("Build should return ErrMultipleBackends; got %v", err)
		}
	})

	t.Run("calls SetOptions on plugins implementing OptionsProvider", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFEWithOpts{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithPluginOptions("fe", map[string]string{"output": "internal/users"}).
			Build()
		assertNoError(t, err)
		if p == nil {
			t.Fatalf("Build should succeed when options are valid")
		}
	})

	t.Run("returns ErrInvalidOptions when SetOptions fails", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().
			WithFrontend(&stubFEWithOpts{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			// "output" is required; supplying nothing triggers
			// ErrMissingRequired inside Decode.
			Build()
		if !errors.Is(err, pipeline.ErrInvalidOptions) {
			t.Fatalf("Build should return ErrInvalidOptions; got %v", err)
		}
	})

	t.Run("writes one diagnostic per validation error", func(t *testing.T) {
		t.Parallel()
		d := diag.New()
		_, _ = pipeline.New().
			WithFrontend(&stubFE{name: "shared"}).
			WithAnnotator(&stubAnn{name: "shared"}).
			WithDiag(d).
			Build()
		// Expected errors: duplicate name + no backend = 2.
		if d.Count(diag.Error) < 2 {
			t.Fatalf("Build should write per-error diagnostics; got %d", d.Count(diag.Error))
		}
	})

	t.Run("aggregates multiple errors via errors.Join", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "shared"}).
			WithAnnotator(&stubAnn{name: "shared"}).
			Build()
		if !errors.Is(err, pipeline.ErrDuplicatePlugin) {
			t.Fatalf("aggregate should match ErrDuplicatePlugin; got %v", err)
		}
		if !errors.Is(err, pipeline.ErrNoBackend) {
			t.Fatalf("aggregate should match ErrNoBackend; got %v", err)
		}
	})

	t.Run("ignores empty plugin names when checking duplicates", func(t *testing.T) {
		t.Parallel()
		// Two plugins reporting the empty string are not considered
		// duplicates — the empty name signals an unnamed stub which
		// the pipeline tolerates here (later milestones may surface
		// it as its own diagnostic).
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: ""}).
			WithAnnotator(&stubAnn{name: ""}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		if errors.Is(err, pipeline.ErrDuplicatePlugin) {
			t.Fatalf("empty names must not collide as duplicates; got %v", err)
		}
	})
}

func TestBuilder_WithDirective(t *testing.T) {
	t.Parallel()

	t.Run("registers schemas on the pipeline's directive.Registry", func(t *testing.T) {
		t.Parallel()
		repo := directive.NewSchema("repo").Build()
		mock := directive.NewSchema("mock").Build()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithDirective(repo, mock).
			Build()
		assertNoError(t, err)
		reg := p.DirectiveRegistry()
		if _, ok := reg.Lookup("repo"); !ok {
			t.Fatalf("registry should contain 'repo'")
		}
		if _, ok := reg.Lookup("mock"); !ok {
			t.Fatalf("registry should contain 'mock'")
		}
	})

	t.Run("variadic and repeated calls accumulate", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithDirective(directive.NewSchema("a").Build()).
			WithDirective(directive.NewSchema("b").Build(), directive.NewSchema("c").Build()).
			Build()
		assertNoError(t, err)
		if got := p.DirectiveRegistry().Names(); len(got) != 3 {
			t.Fatalf("expected 3 registered names; got %v", got)
		}
	})

	t.Run("duplicate schemas return ErrDuplicateDirective", func(t *testing.T) {
		t.Parallel()
		schema := directive.NewSchema("dup").Build()
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithDirective(schema, schema).
			Build()
		if !errors.Is(err, pipeline.ErrDuplicateDirective) {
			t.Fatalf("Build should return ErrDuplicateDirective; got %v", err)
		}
	})
}

func TestBuilder_Build_EmitVersionCompatibility(t *testing.T) {
	t.Parallel()

	t.Run("rejects a plugin whose declared majors omit the current emit major", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().
			WithFrontend(&emitVersionedFE{name: "fe", versions: []string{"99"}}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		if !errors.Is(err, pipeline.ErrIncompatibleEmitVersion) {
			t.Fatalf("Build should return ErrIncompatibleEmitVersion; got %v", err)
		}
	})

	t.Run("accepts a plugin whose declared majors include the current emit major", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().
			WithFrontend(&emitVersionedFE{name: "fe", versions: []string{"1"}}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
	})

	t.Run("plugins not implementing EmitVersioned are assumed compatible", func(t *testing.T) {
		t.Parallel()
		_, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			Build()
		assertNoError(t, err)
	})
}

func TestBuilder_WithParallel(t *testing.T) {
	t.Parallel()

	t.Run("returns the receiver for chaining", func(t *testing.T) {
		t.Parallel()
		b := pipeline.New()
		if out := b.WithParallel(pipeline.PhaseFrontend, pipeline.PhaseAnnotator); out != b {
			t.Fatalf("WithParallel should return the receiver")
		}
	})
}

func TestBuilder_WithManifestPath(t *testing.T) {
	t.Parallel()

	t.Run("returns the receiver for chaining", func(t *testing.T) {
		t.Parallel()
		b := pipeline.New()
		if out := b.WithManifestPath("/tmp/manifest.json"); out != b {
			t.Fatalf("WithManifestPath should return the receiver")
		}
	})
}
