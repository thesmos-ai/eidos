// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sentinel_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	sentinelplugin "go.thesmos.sh/eidos/plugins/generator/sentinel"
	"go.thesmos.sh/eidos/store"
)

// TestConformance runs the framework conformance suites against
// the sentinel plugin. The framework checks pin the static
// contract; the generator suite drives the plugin against
// fixtures covering the empty-package short-circuit, a package
// with Err* sentinels only, a package with custom error types
// only, and the canonical mixed shape.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, sentinelplugin.New())
	})

	t.Run("generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			sentinelplugin.New(),
			[]plugintest.GeneratorFixture{
				{
					Name: "empty package",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return storefixture.New().Build()
					},
				},
				{
					Name: "annotated package with Err* sentinels only",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildSentinelOnlyStore(t)
					},
				},
				{
					Name: "annotated package with a custom error type only",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildErrorTypeOnlyStore(t)
					},
				},
				{
					Name: "annotated package mixing sentinels and error types",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildMixedStore(t)
					},
				},
				{
					Name: "un-annotated package emits nothing",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						// Package without the +gen:sentinel directive
						// must be skipped silently.
						return storefixture.New().
							Package("auth", "example.com/auth").
							Build()
					},
				},
			},
		)
	})

	t.Run("options round-trip", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, sentinelplugin.New(), plugintest.OptionsFixture{
			Valid:      map[string]string{},
			UnknownKey: "no_such_key",
		})
	})

	t.Run("prefix= override is honoured per package", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(
			t,
			sentinelplugin.New(),
			[]plugintest.GeneratorFixture{
				{
					Name: "prefix=custom overrides the default",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildStoreWithPrefixOverride(t, "custom: ")
					},
				},
				{
					Name: "prefix=off disables the prefix subtest",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildStoreWithPrefixOverride(t, "off")
					},
				},
				{
					Name: "prefix= (empty) disables the prefix subtest",
					BuildStore: func(t *testing.T) *store.Store {
						t.Helper()
						return buildStoreWithPrefixOverride(t, "")
					},
				},
			},
		)
	})
}

// buildStoreWithPrefixOverride returns a [store.Store] populated
// with an annotated package whose `+gen:sentinel` directive
// carries the supplied prefix value. Both "off" and the empty
// string disable the prefix subtest; any other value pins the
// prefix the subtest asserts.
func buildStoreWithPrefixOverride(t *testing.T, prefix string) *store.Store {
	t.Helper()
	b := storefixture.New().
		Package("auth", "example.com/auth").
		Variable("ErrFoo", func(vb *storefixture.VariableBuilder) {
			vb.Type(&node.TypeRef{Name: "error"})
		})
	pkg := b.PackageNode()
	pkg.DirectiveList = append(pkg.DirectiveList, storefixture.Directive(
		sentinelplugin.DirectiveName,
		storefixture.KV(sentinelplugin.PrefixKey, prefix),
	))
	return b.Build()
}

// buildSentinelOnlyStore returns a [store.Store] populated with
// one annotated source package declaring two Err* sentinel
// variables but no custom error types — exercises the
// Sentinels-only branch of the rendered template.
func buildSentinelOnlyStore(t *testing.T) *store.Store {
	t.Helper()
	b := storefixture.New().
		Package("auth", "example.com/auth").
		Variable("ErrUnauthorised", func(vb *storefixture.VariableBuilder) {
			vb.Type(&node.TypeRef{Name: "error"})
		}).
		Variable("ErrTokenExpired", func(vb *storefixture.VariableBuilder) {
			vb.Type(&node.TypeRef{Name: "error"})
		})
	annotatePackage(b)
	return b.Build()
}

// buildErrorTypeOnlyStore returns a [store.Store] populated
// with one annotated source package declaring a custom error
// type but no Err* sentinels — exercises the ErrorTypes-only
// branch of the rendered template.
func buildErrorTypeOnlyStore(t *testing.T) *store.Store {
	t.Helper()
	b := storefixture.New().
		Package("auth", "example.com/auth").
		Struct("ValidationError", func(sb *storefixture.StructBuilder) {
			sb.Field("Field", &node.TypeRef{Name: "string"}, nil)
			sb.Field("Reason", &node.TypeRef{Name: "string"}, nil)
			addErrorMethod(sb)
		})
	annotatePackage(b)
	return b.Build()
}

// buildMixedStore returns a [store.Store] populated with one
// annotated source package declaring both Err* sentinels and a
// custom error type — the canonical real-world shape testkit's
// sentinel generator targets.
func buildMixedStore(t *testing.T) *store.Store {
	t.Helper()
	b := storefixture.New().
		Package("auth", "example.com/auth").
		Variable("ErrUnauthorised", func(vb *storefixture.VariableBuilder) {
			vb.Type(&node.TypeRef{Name: "error"})
		}).
		Variable("ErrTokenExpired", func(vb *storefixture.VariableBuilder) {
			vb.Type(&node.TypeRef{Name: "error"})
		}).
		Struct("ValidationError", func(sb *storefixture.StructBuilder) {
			sb.Field("Field", &node.TypeRef{Name: "string"}, nil)
			addErrorMethod(sb)
		})
	annotatePackage(b)
	return b.Build()
}

// annotatePackage attaches a `+gen:sentinel` directive to the
// builder's accumulating package node. The storefixture builder
// has no top-level package-directive setter, so the directive
// list is mutated directly through [storefixture.Builder.PackageNode].
func annotatePackage(b *storefixture.Builder) {
	pkg := b.PackageNode()
	pkg.DirectiveList = append(pkg.DirectiveList, storefixture.Directive(sentinelplugin.DirectiveName))
}

// addErrorMethod attaches an `Error() string` method to the
// struct so [structImplementsError] recognises it as a custom
// error type. The method's body is empty — the test fixtures
// exercise the plugin's source-detection path, not the
// rendered tests' runtime behaviour.
func addErrorMethod(sb *storefixture.StructBuilder) {
	sb.Method(sentinelplugin.ErrorMethodName, func(mb *storefixture.MethodBuilder) {
		mb.Return(&node.TypeRef{Name: "string"})
	})
}

// Silence the "imported and not used" lint when the directive
// package isn't referenced in tests that don't override the
// per-package directive (the storefixture's Directive helper
// uses the import transitively).
var _ = directive.Validate
