// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package demopipe is a black-box pipeline harness for the
// demonstration plugin set. It resolves the canonical
// `eidostest/demoproject` fixture, builds a pipeline from a
// caller-supplied plugin combination, runs the pipeline against
// the fixture, and returns the resulting sink + diagnostics +
// store for assertions.
//
// The harness is the single entry point every demonstration-plugin
// test uses — keeping the fixture path resolution, pipeline wiring,
// and frontend-options plumbing in one place avoids per-test
// duplication and gives every consumer the same view of the
// fixture.
package demopipe

import (
	"maps"
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	frontend_golang "go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// FixtureRoot returns the absolute path to the demoproject fixture
// resolved against this source file's directory via
// [runtime.Caller]. The path is stable regardless of the test's
// working directory — `go test` invocations from any depth in the
// repository tree reach the same fixture.
func FixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("demopipe: runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "demoproject")
}

// FixturePattern is the canonical pattern the harness passes to the
// Go frontend's Load call. It targets every package under the
// `blog` directory of the fixture; harness callers that need a
// narrower pattern (e.g. just `extras`) build their own.
const FixturePattern = "./blog/..."

// RunOptions configures one call to [Run]. The zero value is
// runnable: the harness defaults Sink to an in-memory sink and
// Pattern to [FixturePattern], so a caller wiring a single
// generator only sets the Generators slice.
type RunOptions struct {
	// Annotators are the annotator plugins to register on the
	// pipeline in slice order. Empty slice runs no annotators.
	Annotators []plugin.Annotator

	// Generators are the generator plugins to register on the
	// pipeline in slice order. Empty slice runs no generators.
	Generators []plugin.Generator

	// Backend is the backend plugin to register. Required.
	Backend plugin.Backend

	// Sink captures the backend's output. Defaults to a fresh
	// [sink.NewMemory] when nil.
	Sink sink.Sink

	// Pattern overrides [FixturePattern]. Empty falls back to the
	// canonical pattern.
	Pattern string

	// FrontendOptions extends or overrides the harness's default
	// frontend options. The harness always sets Dir to the
	// fixture root; callers may add other entries (build_tags,
	// include_tests, …) without restating Dir.
	FrontendOptions map[string]string

	// PluginOptions carries per-plugin configuration keyed by
	// plugin name. Each entry is forwarded verbatim to
	// [pipeline.Builder.WithPluginOptions]. Empty map runs every
	// plugin with its built-in defaults.
	PluginOptions map[string]map[string]string

	// Layout selects the routing layout the pipeline uses for the
	// run — either [pipeline.LayoutAlongsideSource] (the framework
	// default; rendered files land beside the originating source)
	// or [pipeline.LayoutCentralised] (rendered files land in a
	// shared directory). Empty defers to the framework default.
	Layout string

	// OutputPackage pins [emit.Target.Package] for every emitted
	// decl in scope and supplies the shared package name centralised
	// layouts require. Empty leaves the per-decl default in place.
	OutputPackage string

	// OutputDir sets the rendered output directory under
	// centralised layout. Ignored under alongside-source. Empty
	// defers to [OutputPackage] when centralised is selected.
	OutputDir string

	// TargetSymbol restricts the run to source decls whose
	// unqualified Name equals the value, or whose QName ends with
	// `.<value>`. Empty disables scoping (every in-scope decl
	// participates). Maps to [pipeline.Builder.WithTargetSymbol].
	TargetSymbol string
}

// Result captures the outcome of a [Run] call. Tests assert
// against the captured outputs, the diagnostics raised during the
// run, and the resulting store for node / emit / metadata
// inspection.
type Result struct {
	// Sink is the destination sink that captured backend output.
	// Tests cast it to its concrete type (typically *sink.Memory)
	// to enumerate written files.
	Sink sink.Sink

	// Diag is the diagnostic sink the pipeline shared across every
	// phase. HasErrors / Diagnostics give per-test visibility.
	Diag *diag.Sink

	// Store is the in-memory store the pipeline populated. Nil
	// when the pipeline never reached the Run path (e.g. Build
	// failed).
	Store *store.Store

	// RunErr is the error returned by [pipeline.Pipeline.Run].
	// Typically [pipeline.ErrRunHadErrors] when any plugin emitted
	// an Error diagnostic; nil otherwise.
	RunErr error
}

// Run builds and runs a pipeline against the demoproject fixture
// per opts and returns the captured outputs. Build failures
// surface as a fatal test error — the harness is for tests that
// assume the pipeline's plugin set is valid; collision diagnostics
// belong in builder-focused tests, not harness-driven ones.
func Run(t *testing.T, opts RunOptions) Result {
	t.Helper()
	if opts.Backend == nil {
		t.Fatalf("demopipe.Run: opts.Backend is required")
	}
	if opts.Sink == nil {
		opts.Sink = sink.NewMemory()
	}
	pattern := opts.Pattern
	if pattern == "" {
		pattern = FixturePattern
	}

	feOpts := frontendOptionsFor(t, opts)
	fe := frontend_golang.New()

	b := pipeline.New().
		WithFrontend(fe).
		WithBackend(opts.Backend).
		WithSink(opts.Sink).
		WithPluginOptions(fe.Name(), feOpts)
	if opts.Layout != "" {
		b = b.WithOutputLayout(opts.Layout)
	}
	if opts.OutputPackage != "" {
		b = b.WithOutputPackage(opts.OutputPackage)
	}
	if opts.OutputDir != "" {
		b = b.WithOutputDir(opts.OutputDir)
	}
	if opts.TargetSymbol != "" {
		b = b.WithTargetSymbol(opts.TargetSymbol)
	}
	for _, a := range opts.Annotators {
		b = b.WithAnnotator(a)
	}
	for _, g := range opts.Generators {
		b = b.WithGenerator(g)
	}
	for name, kv := range opts.PluginOptions {
		b = b.WithPluginOptions(name, kv)
	}

	p, err := b.Build()
	if err != nil {
		t.Fatalf("demopipe.Run: pipeline.Build: %v", err)
	}
	runErr := p.Run(t.Context(), pattern)
	return Result{
		Sink:   opts.Sink,
		Diag:   p.Diag(),
		Store:  p.Store(),
		RunErr: runErr,
	}
}

// frontendOptionsFor merges the harness's default frontend options
// (`dir=<fixture root>`) with caller-supplied overrides. Callers
// that pre-populate Dir win — the harness only fills the slot when
// caller didn't.
func frontendOptionsFor(t *testing.T, opts RunOptions) map[string]string {
	t.Helper()
	out := map[string]string{"dir": FixtureRoot(t)}
	maps.Copy(out, opts.FrontendOptions)
	return out
}
