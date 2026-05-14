// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo_test

import (
	"testing"

	"go.thesmos.sh/eidos/bridge/protogo"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the protogo bridge annotator: the universal
// [plugintest.RunSuite] for stability / role / capability
// contracts, plus the per-role [plugintest.RunAnnotatorSuite]
// for idempotency / frozen-store / diagnostic discipline against
// representative source-side fixtures.
//
// The bridge annotator stamps cross-frontend meta keys on source
// nodes loaded by the proto frontend; for the conformance pass
// we drive it against language-agnostic synthetic stores —
// fixture-level shape suffices for the determinism / frozen-
// store properties the suite asserts.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, protogo.New())
	})

	t.Run("annotator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunAnnotatorSuite(
			t,
			protogo.New(),
			[]plugintest.AnnotatorFixture{
				{
					Name: "empty package",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().Build()
					},
				},
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
	})
}
