// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package debugweaver_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/reference/debugweaver"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// debugweaver. The plugin is a cross-cutting Prebody-slot weaver
// — it contributes statements to existing methods that other
// generators already produced, so the conformance suite's
// fixtures need only assert the contract holds for any source
// shape; the rendered output is exercised end-to-end through the
// pipeline conformance harnesses downstream.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, debugweaver.New())
	})

	t.Run("generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			debugweaver.New(),
			[]plugintest.GeneratorFixture{
				{
					Name: "empty package",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().Build()
					},
				},
				{
					Name: "package with a struct",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Struct("User", nil).
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, debugweaver.New(), plugintest.OptionsFixture{
			Valid: map[string]string{
				"package": "log",
				"func":    "Printf",
				"format":  "debug: %s entered",
			},
			UnknownKey: "no_such_field",
		})
	})
}
