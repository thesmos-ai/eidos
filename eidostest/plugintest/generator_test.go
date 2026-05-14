// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest_test

import (
	"fmt"
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

// TestRunGeneratorSuite_PassesForWellFormedGenerator pins the
// happy path of [plugintest.RunGeneratorSuite]: a generator
// that emits one struct per source struct passes every contract
// — does not panic, leaves source-node counts intact, and
// produces identical emit projections across two equivalent
// runs.
func TestRunGeneratorSuite_PassesForWellFormedGenerator(t *testing.T) {
	t.Parallel()
	plugintest.RunGeneratorSuite(
		t,
		&mirroringGenerator{name: "mirror"},
		[]plugintest.GeneratorFixture{
			{
				Name: "package with one struct",
				BuildStore: func(t *testing.T) *store.Store {
					t.Helper()
					return storefixture.New().
						Struct("User", nil).
						Build()
				},
			},
			{
				Name: "package with three structs",
				BuildStore: func(t *testing.T) *store.Store {
					t.Helper()
					return storefixture.New().
						Struct("User", nil).
						Struct("Order", nil).
						Struct("Invoice", nil).
						Build()
				},
			},
		},
	)
}

// TestRunGeneratorSuite_RejectsPanickingGenerator covers the
// empty-store panic rejection.
func TestRunGeneratorSuite_RejectsPanickingGenerator(t *testing.T) {
	t.Parallel()
	a := &panickingGenerator{name: "panicky"}
	fake := newFakeT()
	plugintest.AssertGenerateEmptyStoreDoesNotPanic(fake, a)
	assertFakeMentions(t, fake, "Generate panicked on empty store")
}

// TestRunGeneratorSuite_RejectsNonDeterministicGenerator pins
// the determinism contract: a generator whose output depends on
// call-count (or any other call-site-varying input) fails the
// two-run comparison.
func TestRunGeneratorSuite_RejectsNonDeterministicGenerator(t *testing.T) {
	t.Parallel()
	s1 := storefixture.New().Struct("User", nil).Build()
	s2 := storefixture.New().Struct("User", nil).Build()
	g := &flappingGenerator{name: "flap"}
	fake := newFakeT()
	plugintest.AssertGenerateIsDeterministic(fake, g, s1, s2)
	assertFakeMentions(t, fake, "generator is not deterministic")
}

// TestRunGeneratorSuite_RejectsSourceMutator pins the
// frozen-source-nodes contract: a generator that mutates the
// source side of the store fails the node-count check.
func TestRunGeneratorSuite_RejectsSourceMutator(t *testing.T) {
	t.Parallel()
	s := storefixture.New().Struct("User", nil).Build()
	g := &sourceMutatingGenerator{name: "mutate"}
	fake := newFakeT()
	plugintest.AssertGenerateLeavesSourceNodesUnchanged(fake, g, s)
	assertFakeMentions(t, fake, "Generate changed source-side node counts")
}

// TestRunGeneratorSuite_FailsOnDuplicateFixtureName pins the
// fixture-name uniqueness contract.
func TestRunGeneratorSuite_FailsOnDuplicateFixtureName(t *testing.T) {
	t.Parallel()
	fixtures := []plugintest.GeneratorFixture{
		{Name: "dup", BuildStore: func(t *testing.T) *store.Store {
			t.Helper()
			return storefixture.New().Build()
		}},
		{Name: "dup", BuildStore: func(t *testing.T) *store.Store {
			t.Helper()
			return storefixture.New().Build()
		}},
	}
	fake := newFakeT()
	captureFatal(func() {
		plugintest.AssertGeneratorFixtureNamesUnique(fake, fixtures)
	})
	assertFakeMentions(t, fake, "duplicate fixture Name")
}

// mirroringGenerator emits a single output [emit.Struct] per
// source [node.Struct], with a deterministic name and target
// derived from the source. Idempotent / deterministic by
// construction — re-running with equivalent inputs produces an
// equivalent emit set.
type mirroringGenerator struct{ name string }

// Name returns the configured identifier.
func (g *mirroringGenerator) Name() string { return g.name }

// Generate copies every source struct into one emit struct in
// a single output package.
func (*mirroringGenerator) Generate(ctx *plugin.GeneratorContext) error {
	structs := ctx.Store.Nodes().Structs().Items()
	if len(structs) == 0 {
		return nil
	}
	pkg := &emit.Package{
		Name: "mirror",
		Path: "example.com/mirror",
	}
	for _, src := range structs {
		pkg.Structs = append(pkg.Structs, &emit.Struct{
			Name:    src.Name + "Mirror",
			Package: pkg.Name,
			Target: emit.Target{
				Dir:      pkg.Path,
				Filename: src.Name + "_mirror.go",
				Package:  pkg.Name,
			},
		})
	}
	if err := ctx.Store.Emit().AddPackage(pkg); err != nil {
		return fmt.Errorf("mirroringGenerator: AddPackage: %w", err)
	}
	return nil
}

// panickingGenerator panics in Generate. Used to verify the
// empty-store probe recovers and reports a contract failure.
type panickingGenerator struct{ name string }

// Name returns the configured identifier.
func (g *panickingGenerator) Name() string { return g.name }

// Generate panics on every call.
func (*panickingGenerator) Generate(_ *plugin.GeneratorContext) error {
	panic("plugintest test: panickingGenerator panicking on purpose") //nolint:forbidigo
}

// flappingGenerator emits a struct whose name embeds a
// per-instance counter, so two distinct generator instances
// produce identical output on equivalent inputs but the *same*
// instance varies its output across two calls. The suite's
// determinism check runs against the same generator instance
// for both passes, so this exhibits the flapping behaviour the
// check is supposed to catch.
type flappingGenerator struct {
	name  string
	count int
}

// Name returns the configured identifier.
func (g *flappingGenerator) Name() string { return g.name }

// Generate emits a struct whose name embeds the per-call
// counter, breaking determinism across runs.
func (g *flappingGenerator) Generate(ctx *plugin.GeneratorContext) error {
	g.count++
	pkg := &emit.Package{Name: "flap", Path: "example.com/flap"}
	pkg.Structs = append(pkg.Structs, &emit.Struct{
		Name:    fmt.Sprintf("Flap%d", g.count),
		Package: pkg.Name,
		Target: emit.Target{
			Dir:      pkg.Path,
			Filename: fmt.Sprintf("flap_%d.go", g.count),
			Package:  pkg.Name,
		},
	})
	if err := ctx.Store.Emit().AddPackage(pkg); err != nil {
		return fmt.Errorf("flappingGenerator: AddPackage: %w", err)
	}
	return nil
}

// sourceMutatingGenerator violates the frozen-source contract
// by appending a synthetic struct to the source side of the
// store during Generate.
type sourceMutatingGenerator struct{ name string }

// Name returns the configured identifier.
func (g *sourceMutatingGenerator) Name() string { return g.name }

// Generate adds a synthetic struct to the first source package
// it sees.
func (*sourceMutatingGenerator) Generate(ctx *plugin.GeneratorContext) error {
	pkgs := ctx.Store.Nodes().Packages().Items()
	if len(pkgs) == 0 {
		return nil
	}
	pkg := pkgs[0]
	if err := ctx.Store.Nodes().Structs().Add("Injected", nil); err != nil {
		return fmt.Errorf("sourceMutatingGenerator: Add: %w", err)
	}
	_ = pkg
	return nil
}
