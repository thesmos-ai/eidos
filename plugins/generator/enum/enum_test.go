// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	enumplugin "go.thesmos.sh/eidos/plugins/generator/enum"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the enum plugin. The framework checks pin the static contract
// (stable Name, role implementation, deterministic Outputs,
// well-formed shape); the generator suite pins the
// determinism / frozen-source / diagnostic-discipline contracts
// across a representative fixture set.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, enumplugin.New())
	})

	t.Run("generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			enumplugin.New(),
			[]plugintest.GeneratorFixture{
				{
					Name: "empty package",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().Build()
					},
				},
				{
					Name: "annotated enum with two variants",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildEnumStore(t, withoutOverrides)
					},
				},
				{
					Name: "annotated enum with a +gen:value override",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildEnumStore(t, withOverride)
					},
				},
				{
					Name: "annotated enum with no variants (skipped with diagnostic)",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().
							Enum("Empty", func(eb *storefixture.EnumBuilder) {
								eb.Directive(storefixture.Directive(enumplugin.DirectiveName))
							}).
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, enumplugin.New(), plugintest.OptionsFixture{
			Valid: map[string]string{
				"strip_prefix":    "true",
				"parse_prefix":    "Parse",
				"sentinel_prefix": "ErrUnknown",
			},
			UnknownKey: "no_such_key",
		})
	})
}

// buildEnumStore returns a [store.Store] populated with one
// source enum named Status declared in `status/status.go`. The
// configure hook tweaks the per-test variant set / directive
// layout. Used by the generator suite, which expects a full
// store; the backend-driven end-to-end acceptance test lives
// outside this package (plugins cannot import backends).
func buildEnumStore(t *testing.T, configure func(*storefixture.EnumBuilder)) *store.Store {
	t.Helper()
	return storefixture.New().
		Package("status", "example.com/status").
		Enum("Status", func(eb *storefixture.EnumBuilder) {
			eb.Pos(position.At("status/status.go", 1, 1))
			eb.Directive(storefixture.Directive(enumplugin.DirectiveName))
			configure(eb)
		}).
		Build()
}

// withoutOverrides configures the canonical two-variant enum
// (`StatusActive`, `StatusInactive`) with no per-variant
// `+gen:value` directives — the default prefix-stripping rule
// resolves both to `"Active"` / `"Inactive"`.
func withoutOverrides(eb *storefixture.EnumBuilder) {
	eb.Variant("StatusActive", "0")
	eb.Variant("StatusInactive", "1")
}

// withOverride adds a third variant whose rendered string-form
// is pinned via `+gen:value pending_review` — exercising the
// override branch of the variant-name resolver.
func withOverride(eb *storefixture.EnumBuilder) {
	eb.Variant("StatusActive", "0")
	eb.Variant("StatusInactive", "1")
	eb.Variant("StatusPending", "2")
	// Reach into the just-appended variant to attach the
	// override directive — the storefixture's Variant signature
	// is flat (no callback), so the directive list is mutated
	// after construction.
	enum := eb.Node()
	pending := enum.Variants[len(enum.Variants)-1]
	pending.DirectiveList = append(pending.DirectiveList, &directive.Directive{
		Name: "value",
		Args: []string{"pending_review"},
	})
}
