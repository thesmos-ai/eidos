// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mockgen_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/reference/mockgen"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the mockgen plugin.
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
					Name: "package with no annotated interfaces",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Interface("Plain", nil).
							Build()
					},
				},
				{
					Name: "package with one mock-annotated interface",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Interface("Reader", func(i *storefixture.InterfaceBuilder) {
								i.Directive(storefixture.Directive("mock"))
							}).
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, mockgen.New(), plugintest.OptionsFixture{
			Valid:      map[string]string{"suffix": "Stub"},
			UnknownKey: "no_such_field",
		})
	})
}

// newPrimed returns a mockgen plugin with schema defaults
// applied.
func newPrimed(t *testing.T) *mockgen.Plugin {
	t.Helper()
	p := mockgen.New()
	if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
		t.Fatalf("mockgen: prime defaults: %v", err)
	}
	return p
}
