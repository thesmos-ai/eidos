// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package protopipe is a black-box pipeline harness for tests
// targeting the protobuf frontend. It mirrors the shape of
// [eidostest/demopipe] (which targets the Go frontend) — same
// run / capture / assert surface, same handling of generators /
// annotators / backend / sink — but registers
// [frontend/protobuf.Frontend] instead of the Go frontend. Tests
// covering proto-input pipelines use this harness so the
// frontend-specific wiring lives in one place.
//
// Callers supply a proto source root via [RunOptions.SourceDir]
// and the harness drives [Frontend.Load] against it. Generator,
// annotator, backend, and sink slots on [RunOptions] are
// optional; the zero-value pipeline (with the harness's
// no-op backend) exercises the frontend in isolation.
package protopipe

import (
	"maps"
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/frontend/protobuf"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// FixtureRoot resolves the absolute path of the testdata fixture
// named `name` under this package's testdata directory. Resolves
// through [runtime.Caller] so the path is stable regardless of the
// test's working directory.
//
// Proto fixtures live in Go-buildable directories so the
// framework's alongside-source layout produces sensible
// `_mock_test.go` paths under the Go toolchain when the rendered
// output lands beside the .proto sources.
func FixtureRoot(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("protopipe: runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// DefaultPattern is the canonical pattern the harness passes to
// the protobuf frontend's Load call: a recursive sweep over the
// configured source root. Tests that need a narrower scope
// (a specific subdirectory or single file) set
// [RunOptions.Pattern] explicitly.
const DefaultPattern = "./..."

// RunOptions configures one call to [Run]. The zero value plus a
// non-empty [SourceDir] is runnable: the harness defaults Sink to
// an in-memory sink and Pattern to [DefaultPattern], so a caller
// that just wants to exercise the frontend supplies only
// SourceDir.
type RunOptions struct {
	// SourceDir is the proto-source search root the frontend
	// passes through its `dir` option. Required — the harness
	// fails the test when empty.
	SourceDir string

	// Pattern overrides [DefaultPattern]. Empty falls back to the
	// canonical recursive form.
	Pattern string

	// FrontendOptions extends or overrides the harness's default
	// frontend options. The harness always pre-populates `dir`
	// from [SourceDir]; entries here override that default and
	// add other slots (`import_paths`, `include_well_known`, …).
	// Per the precedence rule documented on [frontendOptionsFor],
	// entries in [PluginOptions] keyed by the protobuf frontend's
	// name win over the same keys in FrontendOptions.
	FrontendOptions map[string]string

	// Annotators are the annotator plugins to register on the
	// pipeline in slice order. Optional — frontend-only runs
	// leave this nil; runs that bridge proto-derived sources to
	// a target backend register the matching bridge annotator.
	Annotators []plugin.Annotator

	// Generators are the generator plugins to register on the
	// pipeline in slice order. Optional — frontend-only runs
	// leave this nil.
	Generators []plugin.Generator

	// Backend is the backend plugin to register. Optional; the
	// harness falls back to a minimal no-op backend that
	// satisfies the pipeline's build invariants without emitting
	// anything. Tests covering a render pass set this to the
	// real backend.
	Backend plugin.Backend

	// Sink captures the backend's output when [Backend] is set.
	// Defaults to a fresh in-memory sink when nil.
	Sink sink.Sink

	// PluginOptions carries per-plugin configuration keyed by
	// plugin name. Entries are forwarded to the builder verbatim
	// except for the protobuf-frontend entry, which is merged
	// into the frontend's option map per [frontendOptionsFor]'s
	// precedence rule (PluginOptions > FrontendOptions > harness
	// defaults). Empty map runs every plugin with its built-in
	// defaults.
	PluginOptions map[string]map[string]string

	// Cache is the cache instance the pipeline uses. Defaults to
	// [cache.NewNone] when nil. Tests verifying cache
	// consumption pass a recording cache here.
	Cache cache.Cache
}

// Result captures the outcome of a [Run] call.
type Result struct {
	// Sink is the destination sink that captured backend output
	// (or the harness-supplied no-op sink in frontend-only runs).
	Sink sink.Sink

	// Diag is the diagnostic sink the pipeline shared across
	// every phase. Tests assert against [diag.Sink.HasErrors] /
	// [diag.Sink.Diagnostics] for per-plugin diagnostics.
	Diag *diag.Sink

	// Store is the in-memory store the pipeline populated with
	// proto-derived [node.Package] entries — empty when the
	// frontend's source-mapping converter is wired to emit
	// nothing, populated when the converter is fully wired.
	Store *store.Store

	// RunErr is the error returned by [pipeline.Pipeline.Run].
	// Typically [pipeline.ErrRunHadErrors] when any plugin
	// emitted an Error diagnostic; nil otherwise.
	RunErr error
}

// Run builds and runs a pipeline against the supplied proto
// source root per opts and returns the captured outputs. Build
// failures surface as a fatal test error — the harness is for
// tests that assume the pipeline's plugin set is valid.
//
// When [RunOptions.Backend] is nil, the harness registers a
// minimal no-op backend that satisfies the pipeline's build
// invariants without emitting anything. Tests asserting against
// the frontend's diagnostics or store without driving a render
// pass take the no-op default; tests covering rendered output
// set Backend to the real backend.
func Run(t *testing.T, opts RunOptions) Result {
	t.Helper()
	if opts.SourceDir == "" {
		t.Fatalf("protopipe.Run: opts.SourceDir is required")
	}
	if opts.Sink == nil {
		opts.Sink = sink.NewMemory()
	}
	if opts.Cache == nil {
		opts.Cache = cache.NewNone()
	}
	pattern := opts.Pattern
	if pattern == "" {
		pattern = DefaultPattern
	}
	backend := opts.Backend
	if backend == nil {
		backend = &noopBackend{}
	}

	fe := protobuf.New()
	feOpts := frontendOptionsFor(fe.Name(), opts)

	b := pipeline.New().
		WithFrontend(fe).
		WithBackend(backend).
		WithSink(opts.Sink).
		WithCache(opts.Cache).
		WithPluginOptions(fe.Name(), feOpts)
	for _, a := range opts.Annotators {
		b = b.WithAnnotator(a)
	}
	for _, g := range opts.Generators {
		b = b.WithGenerator(g)
	}
	for name, kv := range opts.PluginOptions {
		if name == fe.Name() {
			// Already folded into feOpts above with the documented
			// precedence — skipping prevents the loop from racing
			// the earlier WithPluginOptions call.
			continue
		}
		b = b.WithPluginOptions(name, kv)
	}

	p, err := b.Build()
	if err != nil {
		t.Fatalf("protopipe.Run: pipeline.Build: %v", err)
	}
	runErr := p.Run(t.Context(), pattern)
	return Result{
		Sink:   opts.Sink,
		Diag:   p.Diag(),
		Store:  p.Store(),
		RunErr: runErr,
	}
}

// frontendOptionsFor assembles the option map handed to the protobuf
// frontend via [pipeline.Builder.WithPluginOptions]. The merge
// precedence (last writer wins) is, from lowest to highest:
//
//  1. The harness's default `dir = <SourceDir>` slot.
//  2. Caller-supplied [RunOptions.FrontendOptions] — the harness's
//     shortcut surface for the protobuf frontend.
//  3. Caller-supplied [RunOptions.PluginOptions] keyed by frontendName
//     — the generic per-plugin override path; wins over the harness
//     shortcut when both populate the same key (callers that target
//     the frontend by full plugin-name expect the most explicit slot
//     to win).
//
// The returned map is a fresh allocation; mutating it does not
// affect the caller's [RunOptions].
func frontendOptionsFor(frontendName string, opts RunOptions) map[string]string {
	out := map[string]string{"dir": opts.SourceDir}
	maps.Copy(out, opts.FrontendOptions)
	maps.Copy(out, opts.PluginOptions[frontendName])
	return out
}

// LoadDirect is a lower-level entry point that exercises the
// frontend's [Frontend.Load] surface without building the full
// pipeline. Tests covering the frontend's diagnostic surface
// (proto-editions rejection, parse-clean fixtures) call
// LoadDirect to avoid the pipeline-build invariants noise; tests
// that need annotator / generator interaction call [Run] instead.
//
// The returned [diag.Sink] captures every diagnostic the
// frontend produced; the returned [store.Store] carries every
// proto-derived [node.Package] entry the frontend's converter
// emitted (empty when the converter is wired to emit nothing).
func LoadDirect(t *testing.T, opts RunOptions) Result {
	t.Helper()
	if opts.SourceDir == "" {
		t.Fatalf("protopipe.LoadDirect: opts.SourceDir is required")
	}
	if opts.Sink == nil {
		opts.Sink = sink.NewMemory()
	}
	if opts.Cache == nil {
		opts.Cache = cache.NewNone()
	}
	pattern := opts.Pattern
	if pattern == "" {
		pattern = DefaultPattern
	}
	d := diag.New()
	s := store.New()
	fe := protobuf.New()
	if err := fe.SetOptions(opt.New(fe.OptionsSchema(), frontendOptionsFor(fe.Name(), opts))); err != nil {
		t.Fatalf("protopipe.LoadDirect: SetOptions: %v", err)
	}
	ctx := &plugin.FrontendContext{
		Store:    s,
		Diag:     d,
		Registry: directive.NewRegistry(),
		Parser:   directive.DefaultParser(),
		Cache:    opts.Cache,
		Pattern:  pattern,
	}
	if err := fe.Load(ctx); err != nil {
		t.Fatalf("protopipe.LoadDirect: Load: %v", err)
	}
	return Result{Sink: opts.Sink, Diag: d, Store: s, RunErr: nil}
}

// noopBackend is the harness-internal backend that satisfies the
// pipeline's build invariants without emitting anything. Used
// when the caller doesn't supply a [RunOptions.Backend]; tests
// covering rendered output pass the real backend instead.
type noopBackend struct{}

func (*noopBackend) Name() string                          { return "protopipe.noop" }
func (*noopBackend) Language() string                      { return "proto" }
func (*noopBackend) Render(_ *plugin.BackendContext) error { return nil }
func (*noopBackend) EmitVersions() []string                { return []string{emit.Major()} }
