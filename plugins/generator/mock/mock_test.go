// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mock_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/generator/mock"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the mock plugin: the universal [plugintest.RunSuite], the
// per-role [plugintest.RunGeneratorSuite] with representative
// fixtures, and [plugintest.RunOptionsSuite] for the options
// round-trip.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, mock.New())
	})

	t.Run("generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			mock.New(),
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
					Name: "single +gen:mock interface",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Interface("Searcher", func(i *storefixture.InterfaceBuilder) {
								i.Directive(storefixture.Directive(mock.DirectiveName))
								i.Method("Get", func(m *storefixture.MethodBuilder) {
									m.Param("ctx", &node.TypeRef{Name: "Context", Package: "context"})
									m.Param("key", &node.TypeRef{Name: "string"})
									m.Return(&node.TypeRef{Name: "Record", Package: "x"})
									m.Return(&node.TypeRef{Name: "error"})
								})
							}).
							Build()
					},
				},
				{
					Name: "void-returning method",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Interface("Notifier", func(i *storefixture.InterfaceBuilder) {
								i.Directive(storefixture.Directive(mock.DirectiveName))
								i.Method("Notify", func(m *storefixture.MethodBuilder) {
									m.Param("msg", &node.TypeRef{Name: "string"})
								})
							}).
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, mock.New(), plugintest.OptionsFixture{
			Valid:      map[string]string{"suffix": "Stub", "field_prefix": "Stub"},
			UnknownKey: "no_such_field",
		})
	})
}
