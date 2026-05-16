// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sdk_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/kind"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
	"go.thesmos.sh/eidos/sdk"
)

// TestTypeAliasesPreserveIdentity pins the contract that every
// re-exported type in [sdk] is a Go type alias — not a wrapper.
// Aliases preserve identity, so a value of type [plugin.Plugin]
// is also a value of type [sdk.Plugin] without conversion. The
// checks below assign across the alias boundary; if any alias
// accidentally becomes a wrapper, this test fails to compile
// rather than at runtime.
//
// The deliberately-redundant `var x plugin.X = sdkAlias`
// pattern is the only way to express the identity assertion at
// compile time. staticcheck ST1023 flags the redundancy; we
// suppress it because the redundancy is the test.
//
//nolint:staticcheck // intentional redundant typing — see docblock above
func TestTypeAliasesPreserveIdentity(t *testing.T) {
	t.Parallel()

	t.Run("role interfaces alias to plugin package", func(t *testing.T) {
		t.Parallel()
		var p1 sdk.Plugin
		var p2 plugin.Plugin = p1
		_ = p2
		var f1 sdk.Frontend
		var f2 plugin.Frontend = f1
		_ = f2
		var a1 sdk.Annotator
		var a2 plugin.Annotator = a1
		_ = a2
		var g1 sdk.Generator
		var g2 plugin.Generator = g1
		_ = g2
		var b1 sdk.Backend
		var b2 plugin.Backend = b1
		_ = b2
	})

	t.Run("contexts alias to plugin package", func(t *testing.T) {
		t.Parallel()
		var fc1 sdk.FrontendContext
		var fc2 plugin.FrontendContext = fc1
		_ = fc2
		var ac1 sdk.AnnotatorContext
		var ac2 plugin.AnnotatorContext = ac1
		_ = ac2
		var gc1 sdk.GeneratorContext
		var gc2 plugin.GeneratorContext = gc1
		_ = gc2
		var bc1 sdk.BackendContext
		var bc2 plugin.BackendContext = bc1
		_ = bc2
	})

	t.Run("capability interfaces alias to plugin package", func(t *testing.T) {
		t.Parallel()
		var c1 sdk.CapabilityProvider
		var c2 plugin.CapabilityProvider = c1
		_ = c2
		var d1 sdk.DirectiveProvider
		var d2 plugin.DirectiveProvider = d1
		_ = d2
		var o1 sdk.OptionsProvider
		var o2 plugin.OptionsProvider = o1
		_ = o2
		var v1 sdk.Versioned
		var v2 plugin.Versioned = v1
		_ = v2
	})

	t.Run("Priority aliases to priority package", func(t *testing.T) {
		t.Parallel()
		var p1 sdk.Priority = sdk.GeneratorFoundation
		var p2 priority.Priority = p1
		_ = p2
	})

	t.Run("directive types alias to directive package", func(t *testing.T) {
		t.Parallel()
		var n1 sdk.DirectiveName = "repo"
		var n2 directive.Name = n1
		_ = n2
		var s1 sdk.DirectiveSchema
		var s2 directive.Schema = s1
		_ = s2
		var d1 sdk.Directive
		var d2 directive.Directive = d1
		_ = d2
	})

	t.Run("Kind aliases to kind package", func(t *testing.T) {
		t.Parallel()
		var k1 sdk.Kind = "struct"
		var k2 kind.Kind = k1
		_ = k2
	})
}

// TestPriorityBucketsMatchUnderlying pins that the SDK's
// re-exported bucket constants resolve to exactly the same
// numeric values as the underlying priority package. A typo or
// stale re-export here would silently reorder a plugin's
// scheduling.
func TestPriorityBucketsMatchUnderlying(t *testing.T) {
	t.Parallel()
	pairs := []struct {
		name     string
		got      sdk.Priority
		expected priority.Priority
	}{
		{"AnnotatorShape", sdk.AnnotatorShape, priority.AnnotatorShape},
		{"AnnotatorRefinement", sdk.AnnotatorRefinement, priority.AnnotatorRefinement},
		{"AnnotatorValidation", sdk.AnnotatorValidation, priority.AnnotatorValidation},
		{"GeneratorFoundation", sdk.GeneratorFoundation, priority.GeneratorFoundation},
		{"GeneratorComposition", sdk.GeneratorComposition, priority.GeneratorComposition},
		{"GeneratorCrossCutting", sdk.GeneratorCrossCutting, priority.GeneratorCrossCutting},
		{"GeneratorFinalize", sdk.GeneratorFinalize, priority.GeneratorFinalize},
		{"DefaultPriority", sdk.DefaultPriority, priority.Default},
	}
	for _, pair := range pairs {
		if pair.got != pair.expected {
			t.Errorf("sdk.%s = %d, want %d (priority.%s)",
				pair.name, pair.got, pair.expected, pair.name)
		}
	}
}

// TestDirectiveBuilderProxiesUnderlying drives the SDK's
// NewDirective + builder-option helpers and confirms the
// resulting schema is structurally equivalent to one built
// directly through the underlying directive package. Both call
// chains must compile and produce equivalent schemas — anything
// else means the SDK's re-export drifted from the source.
func TestDirectiveBuilderProxiesUnderlying(t *testing.T) {
	t.Parallel()
	viaFacade := sdk.NewDirective("repo").
		On(sdk.Kind("struct")).
		Positional("variant", sdk.Required(), sdk.OneOf("default", "stub")).
		Describe("Repository directive").
		Build()
	viaUnderlying := directive.NewSchema("repo").
		On(kind.Kind("struct")).
		Positional("variant", directive.Required(), directive.OneOf("default", "stub")).
		Describe("Repository directive").
		Build()
	if viaFacade.Name != viaUnderlying.Name {
		t.Errorf("Schema.Name via façade = %q, via underlying = %q",
			viaFacade.Name, viaUnderlying.Name)
	}
	if len(viaFacade.AppliesTo) != len(viaUnderlying.AppliesTo) {
		t.Errorf("Schema.AppliesTo length differs: facade=%d underlying=%d",
			len(viaFacade.AppliesTo), len(viaUnderlying.AppliesTo))
	}
	if len(viaFacade.PositionalArgs) != len(viaUnderlying.PositionalArgs) {
		t.Errorf("Schema.PositionalArgs length differs: facade=%d underlying=%d",
			len(viaFacade.PositionalArgs), len(viaUnderlying.PositionalArgs))
	}
}
