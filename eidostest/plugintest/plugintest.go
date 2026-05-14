// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package plugintest ships the framework-conformance suite plugin
// authors run against their plugin instances. The suite drives a
// fixed sequence of contract checks the framework relies on at
// runtime — stable Name, role-interface compliance, deterministic
// capability ordering, unique directive names, options
// round-trip — so a plugin author lands those checks once
// without re-discovering the contracts each time.
//
// Behaviour-specific assertions belong in the plugin's own
// test file; this package is the contract layer below them.
//
// Typical usage:
//
//	func TestMyPlugin_Conformance(t *testing.T) {
//	    plugintest.RunSuite(t, myplugin.New())
//	}
package plugintest

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/plugin"
)

// RunSuite runs every standard framework-conformance check
// against p. The checks fail through `t.Errorf` rather than
// `t.Fatalf` so a single failing contract surfaces every
// downstream failure too — the author sees the full report in
// one run instead of chasing one cascade at a time.
func RunSuite(t *testing.T, p plugin.Plugin) {
	t.Helper()
	t.Run("Name returns a non-empty, stable identifier", func(t *testing.T) {
		t.Parallel()
		assertStableName(t, p)
	})
	t.Run("implements at least one of the documented role interfaces", func(t *testing.T) {
		t.Parallel()
		assertImplementsARole(t, p)
	})
	t.Run("CapabilityProvider returns deterministic Provides + Requires", func(t *testing.T) {
		t.Parallel()
		assertCapabilityProviderStability(t, p)
	})
	t.Run("DirectiveProvider schemas declare unique non-empty names", func(t *testing.T) {
		t.Parallel()
		assertDirectiveSchemaUniqueness(t, p)
	})
	t.Run("Versioned reports a non-empty version string", func(t *testing.T) {
		t.Parallel()
		assertVersionedNonEmpty(t, p)
	})
}

// assertStableName pins Name's empty-string and stability
// contracts: every framework caller treats the result as the
// plugin's identifier — diagnostic attribution, cache-key
// composition, manifest entries all use it. The check calls
// Name twice and rejects empty / mismatched outputs.
func assertStableName(t *testing.T, p plugin.Plugin) {
	t.Helper()
	first := p.Name()
	if first == "" {
		t.Errorf("Plugin.Name returned the empty string; framework callers require a stable identifier")
	}
	second := p.Name()
	if first != second {
		t.Errorf("Plugin.Name not stable across calls: first=%q second=%q", first, second)
	}
}

// assertImplementsARole verifies p satisfies at least one of the
// four role interfaces. A plugin that satisfies none is
// effectively dead code — the pipeline never invokes anything
// on it. The check surfaces this as a contract failure rather
// than a silent no-op at pipeline-Build time.
func assertImplementsARole(t *testing.T, p plugin.Plugin) {
	t.Helper()
	if _, ok := any(p).(plugin.Frontend); ok {
		return
	}
	if _, ok := any(p).(plugin.Annotator); ok {
		return
	}
	if _, ok := any(p).(plugin.Generator); ok {
		return
	}
	if _, ok := any(p).(plugin.Backend); ok {
		return
	}
	t.Errorf(
		"plugin %T implements no role interface "+
			"(Frontend / Annotator / Generator / Backend); pipeline would never invoke it",
		p,
	)
}

// assertCapabilityProviderStability pins the deterministic-
// ordering contract: the resolver depends on Provides / Requires
// returning the same sequence across calls so capability-topo
// ordering stays reproducible.
func assertCapabilityProviderStability(t *testing.T, p plugin.Plugin) {
	t.Helper()
	provider, ok := any(p).(plugin.CapabilityProvider)
	if !ok {
		return
	}
	first := provider.Provides()
	second := provider.Provides()
	if !slices.Equal(first, second) {
		t.Errorf("CapabilityProvider.Provides not stable: first=%v second=%v", first, second)
	}
	firstReq := provider.Requires()
	secondReq := provider.Requires()
	if !slices.Equal(firstReq, secondReq) {
		t.Errorf("CapabilityProvider.Requires not stable: first=%v second=%v", firstReq, secondReq)
	}
	for _, cap := range first {
		if cap == "" {
			t.Errorf("CapabilityProvider.Provides contains the empty string; capability labels must be non-empty")
		}
	}
	for _, cap := range firstReq {
		if cap == "" {
			t.Errorf("CapabilityProvider.Requires contains the empty string; capability labels must be non-empty")
		}
	}
}

// assertDirectiveSchemaUniqueness pins the directive-schema
// contract: every declared schema has a non-empty Name, and no
// two schemas in the same plugin share a Name. Duplicates would
// shadow each other at registration time; the framework's
// directive registry rejects them.
func assertDirectiveSchemaUniqueness(t *testing.T, p plugin.Plugin) {
	t.Helper()
	provider, ok := any(p).(plugin.DirectiveProvider)
	if !ok {
		return
	}
	seen := map[directive.Name]struct{}{}
	for _, schema := range provider.Directives() {
		if schema.Name == "" {
			t.Errorf("DirectiveProvider.Directives contains a schema with empty Name")
			continue
		}
		if _, dup := seen[schema.Name]; dup {
			t.Errorf("DirectiveProvider.Directives declares duplicate schema name %q", schema.Name)
			continue
		}
		seen[schema.Name] = struct{}{}
	}
}

// assertVersionedNonEmpty pins the Versioned contract: when a
// plugin opts into cache invalidation via [plugin.Versioned], the
// returned Version must be non-empty so cache keys can compose
// it. Plugins that don't implement Versioned are unaffected.
func assertVersionedNonEmpty(t *testing.T, p plugin.Plugin) {
	t.Helper()
	versioned, ok := any(p).(plugin.Versioned)
	if !ok {
		return
	}
	if versioned.Version() == "" {
		t.Errorf("Versioned.Version returned the empty string; framework cache requires a stable non-empty version")
	}
}
