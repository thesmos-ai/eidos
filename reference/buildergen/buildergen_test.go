// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package buildergen_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/reference/buildergen"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the buildergen plugin: the universal [plugintest.RunSuite], the
// per-role [plugintest.RunGeneratorSuite] with representative
// fixtures, and [plugintest.RunOptionsSuite] for the options
// round-trip.
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
					Name: "package with one builder-annotated struct",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Struct("User", func(s *storefixture.StructBuilder) {
								s.Directive(storefixture.Directive("builder"))
								s.Field("Name", storefixture.Named("string"), nil)
							}).
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, buildergen.New(), plugintest.OptionsFixture{
			Valid:      map[string]string{"suffix": "Constructor"},
			UnknownKey: "no_such_field",
		})
	})
}

// newPrimed returns a buildergen plugin with schema defaults
// applied. The plugin's options surface only takes effect after
// SetOptions decodes through the schema, which in production
// happens at pipeline Build time. The conformance tests skip
// the pipeline so they prime the plugin manually here.
func newPrimed(t *testing.T) *buildergen.Plugin {
	t.Helper()
	p := buildergen.New()
	if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
		t.Fatalf("buildergen: prime defaults: %v", err)
	}
	return p
}
