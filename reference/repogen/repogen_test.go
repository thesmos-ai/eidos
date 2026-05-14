// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package repogen_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the repogen plugin: the universal [plugintest.RunSuite] for
// stability / role / capability contracts, plus the per-role
// [plugintest.RunGeneratorSuite] for determinism / frozen-source
// against representative input fixtures, plus
// [plugintest.RunOptionsSuite] for the options round-trip.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, newPrimed(t))
	})

	t.Run("generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			newPrimed(t),
			[]plugintest.GeneratorFixture{
				{
					Name: "package with no annotated structs",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Struct("Plain", nil).
							Build()
					},
				},
				{
					Name: "package with one repo-annotated struct",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Struct("User", func(s *storefixture.StructBuilder) {
								s.Directive(storefixture.Directive("repo"))
								s.Field("ID", storefixture.Named("string"), nil)
							}).
							Build()
					},
				},
				{
					Name: "package with three repo-annotated structs",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Struct("User", func(s *storefixture.StructBuilder) {
								s.Directive(storefixture.Directive("repo"))
							}).
							Struct("Order", func(s *storefixture.StructBuilder) {
								s.Directive(storefixture.Directive("repo"))
							}).
							Struct("Invoice", func(s *storefixture.StructBuilder) {
								s.Directive(storefixture.Directive("repo"))
							}).
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, repogen.New(), plugintest.OptionsFixture{
			Valid: map[string]string{
				"interface_suffix": "Storage",
				"struct_suffix":    "Storer",
				"naming":           "Pascal",
			},
			UnknownKey: "no_such_field",
		})
	})
}

// newPrimed returns a repogen plugin with schema defaults
// applied. The plugin's options surface (InterfaceSuffix=Repository,
// StructSuffix=Repo, Naming=Pascal) only takes effect after
// SetOptions decodes through the schema, which in production
// happens at pipeline Build time. The conformance tests skip the
// pipeline so they prime the plugin manually here.
func newPrimed(t *testing.T) *repogen.Plugin {
	t.Helper()
	p := repogen.New()
	if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
		t.Fatalf("repogen: prime defaults: %v", err)
	}
	return p
}
