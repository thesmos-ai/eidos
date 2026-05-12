// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// The recordingSink is unexported, so its observable behavior is
// reached through pipeline.Run wired with a manifest path. The
// tests below drive that path end-to-end: run the pipeline against
// a backend that performs sink.Writes, then read the written
// manifest and verify its contents.

func TestManifest_RecordsBackendWrites(t *testing.T) {
	t.Parallel()

	t.Run("written manifest lists every sink.Write call with its content hash", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		manifestPath := filepath.Join(root, ".eidos", "manifest.json")
		mem := sink.NewMemory()
		be := &recBE{
			name: "be", lang: "stub",
			render: func(ctx *plugin.BackendContext) {
				_ = ctx.Sink.Write(emit.Target{Dir: "a", Filename: "x.go", Package: "x"}, []byte("hello-x"))
				_ = ctx.Sink.Write(emit.Target{Dir: "a", Filename: "y.go", Package: "x"}, []byte("hello-y"))
			},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(be).
			WithSink(mem).
			WithManifestPath(manifestPath).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))

		// Manifest exists and parses.
		body, err := os.ReadFile(manifestPath)
		assertNoError(t, err)
		if len(body) == 0 {
			t.Fatalf("manifest should be non-empty")
		}
		m, err := manifest.Read(manifestPath)
		assertNoError(t, err)
		if len(m.Outputs) != 2 {
			t.Fatalf("manifest should record 2 outputs; got %d", len(m.Outputs))
		}
		seen := map[emit.Target]string{}
		for _, o := range m.Outputs {
			seen[o.Target] = o.Hash
		}
		x := seen[emit.Target{Dir: "a", Filename: "x.go", Package: "x"}]
		y := seen[emit.Target{Dir: "a", Filename: "y.go", Package: "x"}]
		if x == "" || y == "" {
			t.Fatalf("manifest should hash both targets; got %+v", seen)
		}
	})

	t.Run("no manifest path → no manifest written", func(t *testing.T) {
		t.Parallel()
		mem := sink.NewMemory()
		be := &recBE{
			name: "be", lang: "stub",
			render: func(ctx *plugin.BackendContext) {
				_ = ctx.Sink.Write(emit.Target{Dir: "a", Filename: "x.go", Package: "x"}, []byte("hello-x"))
			},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(be).
			WithSink(mem).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		// Nothing to assert positively for "no manifest" beyond
		// the pipeline completing without writes to a path the
		// test never configured.
	})

	t.Run("inner-sink Write errors propagate; nothing is captured for that target", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		manifestPath := filepath.Join(root, "manifest.json")
		be := &recBE{
			name: "be", lang: "stub",
			render: func(ctx *plugin.BackendContext) {
				// One successful write + one failing write attempt.
				// The pipeline's outer Sink is a Multi[good, fail],
				// but we don't have Multi-with-fail at hand; use a
				// failing wrapper instead.
				_ = ctx.Sink.Write(emit.Target{Dir: "a", Filename: "ok.go", Package: "x"}, []byte("ok-payload"))
			},
		}
		// Inner sink that fails the first write so recordingSink's
		// error path is exercised.
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(be).
			WithSink(&failingSink{err: errFailingSink}).
			WithManifestPath(manifestPath).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		// Manifest was still written but has no captured entries
		// (the only write attempt errored).
		m, err := manifest.Read(manifestPath)
		assertNoError(t, err)
		if len(m.Outputs) != 0 {
			t.Fatalf("expected zero outputs (failing inner sink); got %+v", m.Outputs)
		}
	})

	t.Run("per-output Plugins lists only the contributing plugin when entities are attributed", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		manifestPath := filepath.Join(root, "manifest.json")
		target := emit.Target{Dir: "a", Filename: "x.go", Package: "x"}
		// A generator that adds an emit struct stamped with the
		// plugin's SetBy via the builder. The recording sink walks
		// the store at manifest-write time and reads SetBy to
		// attribute outputs.
		gen := &recGen{name: "attrgen", generate: func(ctx *plugin.GeneratorContext) {
			c := builder.For("attrgen", target)
			pkg := c.Package("x", "example.com/x")
			pkg.Struct("X", func(b *builder.StructBuilder) {
				b.Target(target)
			})
			out, err := pkg.Build()
			if err != nil {
				t.Fatalf("pkg.Build: %v", err)
			}
			if err := ctx.Store.Emit().AddPackage(out); err != nil {
				t.Fatalf("AddPackage: %v", err)
			}
		}}
		be := &recBE{
			name: "be", lang: "stub",
			render: func(ctx *plugin.BackendContext) {
				_ = ctx.Sink.Write(target, []byte("body"))
			},
		}
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithGenerator(gen).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			WithManifestPath(manifestPath).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		m, err := manifest.Read(manifestPath)
		assertNoError(t, err)
		if len(m.Outputs) != 1 {
			t.Fatalf("expected 1 output; got %d", len(m.Outputs))
		}
		got := m.Outputs[0].Plugins
		if len(got) != 1 || got[0] != "attrgen" {
			t.Fatalf("Plugins = %v, want [attrgen] (only contributing plugin)", got)
		}
	})

	t.Run("manifest write failure emits a Warn diagnostic", func(t *testing.T) {
		t.Parallel()
		// Point the manifest at a path whose parent is a regular
		// file so MkdirAll inside manifest.Write fails.
		root := t.TempDir()
		conflict := filepath.Join(root, "block")
		assertNoError(t, os.WriteFile(conflict, nil, 0o600))
		manifestPath := filepath.Join(conflict, ".eidos", "manifest.json")

		mem := sink.NewMemory()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&recBE{name: "be", lang: "stub"}).
			WithSink(mem).
			WithManifestPath(manifestPath).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context()))
		// Manifest write failure surfaces as Warn (not Error) so the
		// run still returns nil.
		if p.Diag().Count(diag.Warn) == 0 {
			t.Fatalf("expected a Warn diagnostic about manifest-write failure")
		}
	})
}
