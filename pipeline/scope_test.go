// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// TestScope_PerKindFiltering covers the public scope contract:
// when [pipeline.Builder.WithTargetSymbol] is set, every Reader
// range query the pipeline hands to a plugin pre-filters to nodes
// the predicate matches. The test drives a generator that iterates
// every Reader range method against a populated node store and
// asserts each kind respects the scope.
func TestScope_PerKindFiltering(t *testing.T) {
	t.Parallel()

	t.Run("each range query observes only the in-scope decl per kind", func(t *testing.T) {
		t.Parallel()
		// Build a node.Package containing two of every range-exposed
		// kind: one named "Target" (in scope) and one named "Other"
		// (out of scope). The generator below counts the scoped
		// view's items per kind; we expect exactly one of each.
		pkg := &node.Package{
			Name: "x", Path: "example.com/x",
			Files: []*node.File{
				{
					Name:     "Target",
					Path:     "x/target.go",
					BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/target.go"}},
				},
				{
					Name:     "Other",
					Path:     "x/other.go",
					BaseNode: node.BaseNode{SourcePos: position.Pos{File: "x/other.go"}},
				},
			},
			Imports: []*node.Import{
				{Alias: "Target", Path: "example.com/target"},
				{Alias: "Other", Path: "example.com/other"},
			},
			Structs: []*node.Struct{
				{Name: "Target", Package: "example.com/x", Fields: []*node.Field{
					{Name: "Target"}, {Name: "Other"},
				}, Methods: []*node.Method{
					{Name: "Target"}, {Name: "Other"},
				}},
				{Name: "Other", Package: "example.com/x"},
			},
			Interfaces: []*node.Interface{
				{Name: "Target", Package: "example.com/x"},
				{Name: "Other", Package: "example.com/x"},
			},
			Functions: []*node.Function{
				{Name: "Target", Package: "example.com/x"},
				{Name: "Other", Package: "example.com/x"},
			},
			Variables: []*node.Variable{
				{Name: "Target", Package: "example.com/x"},
				{Name: "Other", Package: "example.com/x"},
			},
			Constants: []*node.Constant{
				{Name: "Target", Package: "example.com/x"},
				{Name: "Other", Package: "example.com/x"},
			},
			Enums: []*node.Enum{
				{Name: "Target", Package: "example.com/x", Variants: []*node.EnumVariant{
					{Name: "Target"}, {Name: "Other"},
				}},
				{Name: "Other", Package: "example.com/x", Variants: []*node.EnumVariant{
					{Name: "Z"},
				}},
			},
			Aliases: []*node.Alias{
				{Name: "Target", Package: "example.com/x"},
				{Name: "Other", Package: "example.com/x"},
			},
		}
		counts := map[string]int{}
		gen := &slotContributingGen{
			name:   "scoper",
			suffix: "_gen.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				counts["packages"] = len(ctx.Reader.Packages().Slice())
				counts["files"] = len(ctx.Reader.Files().Slice())
				counts["imports"] = len(ctx.Reader.Imports().Slice())
				counts["structs"] = len(ctx.Reader.Structs().Slice())
				counts["interfaces"] = len(ctx.Reader.Interfaces().Slice())
				counts["methods"] = len(ctx.Reader.Methods().Slice())
				counts["fields"] = len(ctx.Reader.Fields().Slice())
				counts["functions"] = len(ctx.Reader.Functions().Slice())
				counts["variables"] = len(ctx.Reader.Variables().Slice())
				counts["constants"] = len(ctx.Reader.Constants().Slice())
				counts["enums"] = len(ctx.Reader.Enums().Slice())
				counts["variants"] = len(ctx.Reader.EnumVariants().Slice())
				counts["aliases"] = len(ctx.Reader.Aliases().Slice())
				return nil
			},
		}
		fe := &nodePackageFE{name: "fe", pkg: pkg}
		d := diag.New()
		p, err := pipeline.New().
			WithFrontend(fe).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithDiag(d).
			WithTargetSymbol("Target").
			Build()
		assertNoError(t, err)
		if err := p.Run(t.Context(), "x"); err != nil {
			t.Fatalf("Run: %v; diags: %+v", err, d.Diagnostics())
		}
		// The lone Package in the fixture is named "x", which the
		// scope predicate rejects, so the Packages count is zero
		// rather than one — confirming the predicate fires on
		// structural kinds too.
		if got := counts["packages"]; got != 0 {
			t.Errorf("packages scoped count = %d, want 0", got)
		}
		// Each other kind should show exactly one in-scope entry.
		for _, kind := range []string{
			"files", "imports", "structs", "interfaces", "methods",
			"fields", "functions", "variables", "constants", "enums",
			"variants", "aliases",
		} {
			if got := counts[kind]; got != 1 {
				t.Errorf("%s scoped count = %d, want 1", kind, got)
			}
		}
	})
}

// TestScope_EmptySymbol pins the contract: passing the empty
// string to [pipeline.Builder.WithTargetSymbol] disables the
// scope filter entirely; every node participates regardless of
// Name.
func TestScope_EmptySymbol(t *testing.T) {
	t.Parallel()

	t.Run("empty symbol disables scope; every Struct is observed", func(t *testing.T) {
		t.Parallel()
		pkg := &node.Package{
			Name: "x", Path: "example.com/x",
			Structs: []*node.Struct{
				{Name: "Alpha", Package: "example.com/x"},
				{Name: "Beta", Package: "example.com/x"},
				{Name: "Gamma", Package: "example.com/x"},
			},
		}
		var observed int
		gen := &slotContributingGen{
			name:   "obs",
			suffix: "_gen.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				observed = len(ctx.Reader.Structs().Slice())
				return nil
			},
		}
		p, err := pipeline.New().
			WithFrontend(&nodePackageFE{name: "fe", pkg: pkg}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithTargetSymbol("").
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if observed != 3 {
			t.Fatalf("observed = %d, want 3 (empty symbol = no filter)", observed)
		}
	})
}

// TestScope_QualifiedForm pins the disambiguation contract:
// passing `pkg.Foo` to [pipeline.Builder.WithTargetSymbol] selects
// only the Foo decl in package pkg, leaving same-named decls in
// other packages out of scope.
func TestScope_QualifiedForm(t *testing.T) {
	t.Parallel()

	t.Run("qualified pkg.Name selects only the matching package's decl", func(t *testing.T) {
		t.Parallel()
		pkgA := &node.Package{
			Name: "a", Path: "example.com/a",
			Structs: []*node.Struct{
				{Name: "Foo", Package: "example.com/a"},
			},
		}
		pkgB := &node.Package{
			Name: "b", Path: "example.com/b",
			Structs: []*node.Struct{
				{Name: "Foo", Package: "example.com/b"},
				{Name: "Bar", Package: "example.com/b"},
			},
		}
		var observedQNames []string
		gen := &slotContributingGen{
			name:   "obs",
			suffix: "_gen.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				ctx.Reader.Structs().Each(func(s *node.Struct) {
					observedQNames = append(observedQNames, s.QName())
				})
				return nil
			},
		}
		p, err := pipeline.New().
			WithFrontend(&multiNodePackageFE{name: "fe", pkgs: []*node.Package{pkgA, pkgB}}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithTargetSymbol("b.Foo").
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if len(observedQNames) != 1 || observedQNames[0] != "example.com/b.Foo" {
			t.Fatalf("scoped Structs = %v, want [example.com/b.Foo] only", observedQNames)
		}
	})

	t.Run("bare Name selects every matching decl across packages", func(t *testing.T) {
		t.Parallel()
		pkgA := &node.Package{
			Name: "a", Path: "example.com/a",
			Structs: []*node.Struct{
				{Name: "Foo", Package: "example.com/a"},
			},
		}
		pkgB := &node.Package{
			Name: "b", Path: "example.com/b",
			Structs: []*node.Struct{
				{Name: "Foo", Package: "example.com/b"},
				{Name: "Bar", Package: "example.com/b"},
			},
		}
		var observedQNames []string
		gen := &slotContributingGen{
			name:   "obs",
			suffix: "_gen.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				ctx.Reader.Structs().Each(func(s *node.Struct) {
					observedQNames = append(observedQNames, s.QName())
				})
				return nil
			},
		}
		p, err := pipeline.New().
			WithFrontend(&multiNodePackageFE{name: "fe", pkgs: []*node.Package{pkgA, pkgB}}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithTargetSymbol("Foo").
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if len(observedQNames) != 2 {
			t.Fatalf("scoped Structs = %v, want both Foos across packages", observedQNames)
		}
	})
}

// TestScope_DirectLookupBypass pins the spec contract: direct
// lookups (e.g. [store.Reader] underlying bucket access by qname)
// bypass the scope predicate so plugins can still resolve specific
// cross-references during a scoped run.
func TestScope_DirectLookupBypass(t *testing.T) {
	t.Parallel()

	t.Run("ByQName retrieves an out-of-scope decl during a scoped run", func(t *testing.T) {
		t.Parallel()
		pkg := &node.Package{
			Name: "x", Path: "example.com/x",
			Structs: []*node.Struct{
				{Name: "InScope", Package: "example.com/x"},
				{Name: "Other", Package: "example.com/x"},
			},
		}
		var direct *node.Struct
		var rangeCount int
		gen := &slotContributingGen{
			name:   "obs",
			suffix: "_gen.go",
			contribute: func(ctx *plugin.GeneratorContext) error {
				rangeCount = len(ctx.Reader.Structs().Slice())
				// Direct ByQName via the store reaches around the
				// scoped Reader's range filter — the contract for
				// "I want this specific cross-reference".
				other, _ := ctx.Store.Nodes().Structs().ByQName("example.com/x.Other")
				direct = other
				return nil
			},
		}
		p, err := pipeline.New().
			WithFrontend(&nodePackageFE{name: "fe", pkg: pkg}).
			WithGenerator(gen).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			WithTargetSymbol("InScope").
			Build()
		assertNoError(t, err)
		assertNoError(t, p.Run(t.Context(), "x"))
		if rangeCount != 1 {
			t.Fatalf("scoped range = %d, want 1 (only InScope visible)", rangeCount)
		}
		if direct == nil || direct.Name != "Other" {
			t.Fatalf(
				"ByQName direct lookup should return Other regardless of scope; got %+v",
				direct,
			)
		}
	})
}

// scopeReaderDirect interface assertion — the [store.Reader]'s
// Store accessor exposes the underlying [store.Store] so direct
// lookups bypass scope.
var _ interface {
	Store() *store.Store
} = (*store.Reader)(nil)
