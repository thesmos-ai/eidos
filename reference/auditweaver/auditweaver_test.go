// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package auditweaver_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/reference/auditweaver"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// auditweaver. Cross-cutting weavers operate on emit graphs
// other generators populate, so the suite's per-fixture
// generator probes verify only the contract surface.
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
		plugintest.RunOptionsSuite(t, auditweaver.New(), plugintest.OptionsFixture{
			Valid: map[string]string{
				"package": "log",
				"func":    "Printf",
				"format":  "audit: %s",
			},
			UnknownKey: "no_such_field",
		})
	})
}

// newPrimed returns an auditweaver plugin with schema defaults
// applied.
func newPrimed(t *testing.T) *auditweaver.Plugin {
	t.Helper()
	p := auditweaver.New()
	if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
		t.Fatalf("auditweaver: prime defaults: %v", err)
	}
	return p
}
