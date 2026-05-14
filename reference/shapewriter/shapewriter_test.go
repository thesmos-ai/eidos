// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shapewriter_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/reference/shapewriter"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the shapewriter plugin: the universal [plugintest.RunSuite] for
// stability / role / capability contracts, plus the per-role
// [plugintest.RunAnnotatorSuite] for idempotency / frozen-store
// / diagnostic discipline against representative source fixtures.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, shapewriter.New())
	})

	t.Run("annotator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunAnnotatorSuite(
			t,
			shapewriter.New(),
			[]plugintest.AnnotatorFixture{
				{
					Name: "package with no relevant structs",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Struct("Plain", nil).
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
	})
}
