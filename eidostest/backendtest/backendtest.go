// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package backendtest ships the emit-injection harness backend
// authors use to drive their backend against a pre-built emit
// graph and assert on the rendered output. The harness skips
// frontend / annotator / generator phases entirely; the caller
// supplies an [emit.Package] (or several) and the backend
// renders them through its template surface.
//
// Typical usage:
//
//	func TestMyBackend(t *testing.T) {
//	    result := backendtest.Run(t, backendtest.RunOptions{
//	        Backend: mybackend.New(),
//	        EmitPackages: []*emit.Package{
//	            {Name: "x", Path: "x", Structs: []*emit.Struct{
//	                {Name: "X", Target: emit.Target{
//	                    Dir: "x", Filename: "x.go", Package: "x",
//	                }},
//	            }},
//	        },
//	    })
//	    if result.Diag.HasErrors() {
//	        t.Fatalf("render diagnostics: %+v", result.Diag.Diagnostics())
//	    }
//	    // assert on result.Sink contents...
//	}
//
// Decls inside each [emit.Package] must carry their
// [emit.Target] pre-populated — the harness skips the routing
// layer that would normally derive Target during the layout
// phase. Callers exercising routing decisions belong in
// pipeline-level tests; this harness exists for backend-
// internal contracts (template selection, import resolution,
// formatting, slot composition).
package backendtest

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// RunOptions configures one [Run] call.
type RunOptions struct {
	// Backend is the backend instance under test. Required.
	Backend plugin.Backend

	// EmitPackages are the pre-built emit graphs the backend
	// renders. Each contained decl must carry its [emit.Target]
	// pre-populated; the harness skips the routing layer.
	EmitPackages []*emit.Package

	// Plugins is the set of plugin instances the backend sees on
	// [plugin.BackendContext.Plugins] / [plugin.BackendContext.Ordered].
	// Used for SetBy attribution and TemplateProvider discovery.
	// The backend itself is appended to this slice automatically;
	// callers don't include it.
	Plugins []plugin.Plugin

	// Sink captures the backend's output. Defaults to a fresh
	// [sink.NewMemory] when nil. Tests inspect the result's Sink
	// after [Run] returns.
	Sink sink.Sink

	// SourcesOverride pins the literal entries the backend stamps
	// into rendered files' `Source:` header line. Empty (the
	// default) lets the backend derive sources from each emit
	// node's origin.
	SourcesOverride []string

	// Command pins the literal text the backend stamps into the
	// `Command:` header line. Empty (the default) leaves the
	// backend's own fallback in place.
	Command string
}

// Result captures the outcome of a [Run] call. Tests assert
// against [Result.Sink] for rendered content + [Result.Diag] for
// any diagnostics the backend raised.
type Result struct {
	// Sink is the destination sink that captured backend output.
	// Tests cast it to its concrete type (typically [*sink.Memory])
	// to enumerate written files.
	Sink sink.Sink

	// Diag is the diagnostic sink the backend wrote through.
	Diag *diag.Sink

	// Store is the in-memory store the harness seeded with the
	// caller's emit packages plus the byTarget index the backend
	// reads.
	Store *store.Store
}

// Run drives opts.Backend against the supplied emit packages and
// returns the captured Sink + Diag for assertions. Build-time
// failures (a nil Backend, an emit package the store rejects)
// surface as [testing.T.Fatalf] — the harness is for tests that
// assume the backend's render path is correctly wired.
func Run(t *testing.T, opts RunOptions) Result {
	t.Helper()
	if opts.Backend == nil {
		t.Fatalf("backendtest.Run: opts.Backend is required")
	}
	if opts.Sink == nil {
		opts.Sink = sink.NewMemory()
	}
	s := store.New()
	for _, pkg := range opts.EmitPackages {
		if err := s.Emit().AddPackage(pkg); err != nil {
			t.Fatalf("backendtest.Run: seed emit package %q: %v", pkg.Name, err)
		}
	}
	// Rebuild the byTarget index so the backend's grouping pass
	// sees the freshly-seeded decls. The pipeline does this after
	// the Layout phase; in this harness the caller pre-populated
	// Target on every decl, so the index is computed from those
	// values directly.
	s.Emit().RebuildByTarget()
	d := diag.New()
	plugins := append([]plugin.Plugin(nil), opts.Plugins...)
	plugins = append(plugins, opts.Backend)
	ctx := &plugin.BackendContext{
		Store:           s,
		Reader:          store.NewReader(s),
		Diag:            d,
		Sink:            opts.Sink,
		Lang:            opts.Backend.Language(),
		Plugins:         plugins,
		Ordered:         plugins,
		Command:         opts.Command,
		SourcesOverride: opts.SourcesOverride,
	}
	if err := opts.Backend.Render(ctx); err != nil {
		t.Fatalf("backendtest.Run: Backend.Render: %v", err)
	}
	return Result{Sink: opts.Sink, Diag: d, Store: s}
}
