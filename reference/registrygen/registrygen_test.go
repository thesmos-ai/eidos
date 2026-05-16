// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package registrygen_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/reference/registrygen"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// registrygen. Cross-cutting plugins (Generator + emits a
// per-package init-time registration file) verify both the
// universal framework contracts and the per-role determinism /
// frozen-source / diagnostic-discipline contracts.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, registrygen.New())
	})

	t.Run("generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			registrygen.New(),
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
							Struct("Plain", nil).
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, registrygen.New(), plugintest.OptionsFixture{
			Valid: map[string]string{
				"register_package": "log",
				"register_func":    "Print",
			},
			UnknownKey: "no_such_field",
		})
	})
}
