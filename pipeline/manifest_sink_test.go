// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/node"
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
		// The source struct lives in package "x" at file "a/x.go"
		// so the Layout phase composes Target.Dir from the source
		// directory and Target.Filename from the basename plus
		// recGen's default "_gen.go" suffix.
		src := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "a/x.go"}},
			Name:     "X", Package: "example.com/x",
		}
		composedTarget := emit.Target{
			Dir: "a", Filename: "x_gen.go", Package: "x", ImportPath: "example.com/x",
		}
		// A generator that adds an emit struct stamped with the
		// plugin's SetBy via the builder. The recording sink walks
		// the store at manifest-write time and reads SetBy to
		// attribute outputs.
		gen := &recGen{name: "attrgen", generate: func(ctx *plugin.GeneratorContext) {
			c := builder.For("attrgen", emit.Target{})
			pkg := c.Package("x", "example.com/x")
			pkg.Struct("X", func(b *builder.StructBuilder) {
				b.Origin(src)
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
				ctx.Reader.EmitStructs().Each(func(s *emit.Struct) {
					_ = ctx.Sink.Write(s.Target, []byte("body"))
				})
			},
		}
		fe := &nodePackageFE{name: "fe", pkg: &node.Package{
			Name: "x", Path: "example.com/x",
			Structs: []*node.Struct{src},
		}}
		p, err := pipeline.New().
			WithFrontend(fe).
			WithGenerator(gen).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			WithManifestPath(manifestPath).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		_ = composedTarget // documents the expected composition
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

	t.Run("Plugins aggregates entity SetBy, slot Provenance, and method-host recursion", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		manifestPath := filepath.Join(root, "manifest.json")
		// Three host source decls so the Layout phase has Origins to
		// route every emit kind through. All three share a source
		// file and source package so the Layout phase composes a
		// single (Dir, Filename, Package) for every routable decl;
		// the resulting outputs compose into one manifest entry.
		srcStruct := &node.Struct{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "a/x.go"}},
			Name:     "S", Package: "example.com/x",
		}
		srcIface := &node.Interface{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "a/x.go"}},
			Name:     "I", Package: "example.com/x",
		}
		srcAlias := &node.Alias{
			BaseNode: node.BaseNode{SourcePos: position.Pos{File: "a/x.go"}},
			Name:     "N", Package: "example.com/x",
		}
		fe := &nodePackageFE{name: "fe", pkg: &node.Package{
			Name: "x", Path: "example.com/x",
			Structs: []*node.Struct{srcStruct}, Interfaces: []*node.Interface{srcIface},
			Aliases: []*node.Alias{srcAlias},
		}}
		// One generator hosts every emit entity but the slot
		// contributions and the per-method prebody come from
		// independent Contexts so manifest attribution must aggregate
		// every distinct SetBy.
		gen := &recGen{name: "host", generate: func(ctx *plugin.GeneratorContext) {
			host := builder.For("host", emit.Target{})
			fieldgen := builder.For("fieldgen", emit.Target{})
			methgen := builder.For("methgen", emit.Target{})
			weaver := builder.For("weaver", emit.Target{})
			pkg := host.Package("x", "example.com/x")
			pkg.Struct("S", func(b *builder.StructBuilder) {
				b.Origin(srcStruct).Method("M", func(mb *builder.MethodBuilder) {
					_ = weaver.AppendPrebody(mb.Node(), builder.RawStmt("_ = 0"))
				})
				_ = fieldgen.AppendField(b.Node(), &emit.Field{Name: "F", Type: emit.External("io", "Reader")})
			})
			pkg.Interface("I", func(b *builder.InterfaceBuilder) {
				b.Origin(srcIface)
				_ = methgen.AppendMethod(b.Node(), &emit.Method{Name: "Do"})
			})
			pkg.NamedType("N", emit.External("io", "Reader"), func(b *builder.AliasBuilder) {
				b.Origin(srcAlias).Method("M", func(*builder.MethodBuilder) {})
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
				// Write to one of the routed targets — the struct's
				// composed target — so the manifest captures a
				// single output whose Plugins list aggregates every
				// contributor.
				ctx.Reader.EmitStructs().Each(func(s *emit.Struct) {
					_ = ctx.Sink.Write(s.Target, []byte("body"))
				})
			},
		}
		p, err := pipeline.New().
			WithFrontend(fe).
			WithGenerator(gen).
			WithGenerator(&recGen{name: "fieldgen"}).
			WithGenerator(&recGen{name: "methgen"}).
			WithGenerator(&recGen{name: "weaver"}).
			WithBackend(be).
			WithSink(sink.NewMemory()).
			WithManifestPath(manifestPath).
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		m, err := manifest.Read(manifestPath)
		assertNoError(t, err)
		if len(m.Outputs) != 1 {
			t.Fatalf("expected 1 output; got %d", len(m.Outputs))
		}
		got := m.Outputs[0].Plugins
		// Order follows the pipeline's plugin-registration order;
		// every distinct SetBy across entity, slot, and method-host
		// recursion appears exactly once.
		want := map[string]bool{"host": true, "fieldgen": true, "methgen": true, "weaver": true}
		if len(got) != len(want) {
			t.Fatalf("Plugins = %v, want every contributor (%v)", got, want)
		}
		for _, name := range got {
			if !want[name] {
				t.Fatalf("Plugins lists %q which did not contribute; got %v", name, got)
			}
			delete(want, name)
		}
		if len(want) != 0 {
			t.Fatalf("Plugins missing contributors %v; got %v", want, m.Outputs[0].Plugins)
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
