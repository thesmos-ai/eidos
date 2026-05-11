// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package testpipe

import (
	"testing"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// Builder is the test-tuned analogue of [pipeline.Builder]. It
// forwards every registration call into the underlying pipeline
// builder and seeds shared defaults so that the typical test path is
// "register plugins, Build, Run" — no manual sink or diag wiring.
//
// Defaults applied at New time:
//
//   - A fresh [sink.Memory] is supplied as the destination sink. The
//     captured bytes drive every per-file assertion the [Pipeline]
//     exposes. Tests that need a different sink may call
//     [Builder.WithSink] to override; in that case file-level
//     assertions become unavailable for files routed away from the
//     memory sink.
//   - A fresh [diag.Sink] is supplied so every test starts from a
//     clean diagnostic state.
//   - The cache defaults to [cache.NewNone] — incremental caching
//     is irrelevant for in-process tests and disabling it keeps
//     assertions hermetic.
type Builder struct {
	t     testing.TB
	inner *pipeline.Builder
	sink  *sink.Memory
	diag  *diag.Sink
}

// New starts a Builder bound to tb. Build- and run-time failures
// call tb.Fatalf, so callers do not need to thread error handling
// through every helper invocation.
func New(tb testing.TB) *Builder {
	tb.Helper()
	mem := sink.NewMemory()
	d := diag.New()
	return &Builder{
		t:     tb,
		inner: pipeline.New().WithSink(mem).WithDiag(d).WithCache(cache.NewNone()),
		sink:  mem,
		diag:  d,
	}
}

// WithFrontend registers a frontend on the underlying pipeline.
func (b *Builder) WithFrontend(p plugin.Frontend) *Builder {
	b.inner.WithFrontend(p)
	return b
}

// WithAnnotator registers an annotator on the underlying pipeline.
func (b *Builder) WithAnnotator(p plugin.Annotator) *Builder {
	b.inner.WithAnnotator(p)
	return b
}

// WithGenerator registers a generator on the underlying pipeline.
func (b *Builder) WithGenerator(p plugin.Generator) *Builder {
	b.inner.WithGenerator(p)
	return b
}

// WithBackend registers the pipeline's backend.
func (b *Builder) WithBackend(p plugin.Backend) *Builder {
	b.inner.WithBackend(p)
	return b
}

// WithDirective registers one or more directive schemas with the
// pipeline's directive registry. Mirrors [pipeline.Builder.WithDirective].
func (b *Builder) WithDirective(schemas ...directive.Schema) *Builder {
	b.inner.WithDirective(schemas...)
	return b
}

// WithPluginOptions supplies typed options for the named plugin.
// Mirrors [pipeline.Builder.WithPluginOptions].
func (b *Builder) WithPluginOptions(name string, kv map[string]string) *Builder {
	b.inner.WithPluginOptions(name, kv)
	return b
}

// WithSink replaces the default in-memory sink. After WithSink the
// per-file assertion methods on [Pipeline] cannot observe files
// routed away from the memory sink; tests that need both real
// destinations and assertions should combine the user sink with the
// default memory sink via [sink.NewMulti].
func (b *Builder) WithSink(s sink.Sink) *Builder {
	b.inner.WithSink(s)
	return b
}

// WithCache replaces the default cache.
func (b *Builder) WithCache(c cache.Cache) *Builder {
	b.inner.WithCache(c)
	return b
}

// WithVerbose enables verbose-mode diagnostics for the underlying
// pipeline.
func (b *Builder) WithVerbose(v bool) *Builder {
	b.inner.WithVerbose(v)
	return b
}

// WithParallel opts the underlying pipeline's phases into parallel
// execution.
func (b *Builder) WithParallel(phases ...pipeline.Phase) *Builder {
	b.inner.WithParallel(phases...)
	return b
}

// Build finalises the underlying pipeline. On any Build error the
// method calls t.Fatalf — tests rarely need to assert against
// builder validation errors, so the default is to fail loudly.
// Tests that want to inspect a build error should drive
// [pipeline.Builder] directly.
func (b *Builder) Build() *Pipeline {
	b.t.Helper()
	inner, err := b.inner.Build()
	if err != nil {
		b.t.Fatalf("testpipe: build failed: %v", err)
	}
	return &Pipeline{
		t:     b.t,
		inner: inner,
		sink:  b.sink,
		diag:  b.diag,
	}
}
