// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugins/generator/builder"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the builder plugin: the universal [plugintest.RunSuite], the
// per-role [plugintest.RunGeneratorSuite] with representative
// fixtures, and [plugintest.RunOptionsSuite] for the options
// round-trip.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, builder.New())
	})

	t.Run("generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			builder.New(),
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
					Name: "single +gen:builder struct with exported fields",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Struct("User", func(s *storefixture.StructBuilder) {
								s.Directive(storefixture.Directive(builder.DirectiveName))
								s.Field("Name", storefixture.Named("string"), nil)
								s.Field("Email", storefixture.Named("string"), nil)
							}).
							Build()
					},
				},
				{
					Name: "struct with mixed exported and unexported fields",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Struct("Account", func(s *storefixture.StructBuilder) {
								s.Directive(storefixture.Directive(builder.DirectiveName))
								s.Field("ID", storefixture.Named("string"), nil)
								s.Field("internalTag", storefixture.Named("string"), nil)
							}).
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, builder.New(), plugintest.OptionsFixture{
			Valid:      map[string]string{"suffix": "Constructor", "setter_prefix": "Set"},
			UnknownKey: "no_such_field",
		})
	})
}
