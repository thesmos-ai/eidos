// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mockrecord_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/plugins/generator/mock"
	"go.thesmos.sh/eidos/plugins/generator/mockrecord"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the mockrecord plugin. Generator-suite fixtures pre-populate
// the emit graph by running the mock plugin first — mockrecord
// is a cross-cutter that decorates mock output via the
// `mock.iface` meta key, so the fixtures must reflect that
// pipeline shape.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, mockrecord.New())
	})

	t.Run("generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			mockrecord.New(),
			[]plugintest.GeneratorFixture{
				{
					Name: "package with no mock-annotated interfaces",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Interface("Plain", nil).
							Build()
					},
				},
				{
					Name: "mock plugin output present",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildStoreWithMockOutput(t)
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, mockrecord.New(), plugintest.OptionsFixture{
			Valid:      map[string]string{"field_suffix": "History"},
			UnknownKey: "no_such_field",
		})
	})
}

// buildStoreWithMockOutput builds a store carrying one
// `+gen:mock` source interface plus the mock plugin's emit
// output. mockrecord reads emit-side mock structs (via
// [mock.MetaIface]), so the generator-suite's determinism and
// no-panic checks need a store whose emit graph reflects what
// mockrecord sees in a real pipeline.
func buildStoreWithMockOutput(t *testing.T) *store.Store {
	t.Helper()
	s := storefixture.New().
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
	ctx := &plugin.GeneratorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   diag.New(),
	}
	if err := mock.New().Generate(ctx); err != nil {
		t.Fatalf("mock.Generate(setup): %v", err)
	}
	return s
}
