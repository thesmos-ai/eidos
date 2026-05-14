// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/plugin"
)

// RunSuite runs every framework-conformance check applicable to
// the plugin's declared capabilities and roles. The checks pin
// the invariants the pipeline relies on at registration / build
// time: stable [plugin.Plugin.Name], role-interface compliance,
// deterministic [plugin.CapabilityProvider] ordering, unique
// [plugin.DirectiveProvider] schema names, well-formed
// [plugin.EmitVersioned] entries, stable [plugin.NodesOnly]
// declaration, and stable [plugin.FilenameProvider.FilenameSuffix]
// per language.
//
// Per-role contract checks (idempotency, determinism, diagnostic
// discipline) belong on the role-specific suites: see
// [RunAnnotatorSuite], [RunGeneratorSuite], [RunBackendSuite],
// [RunFrontendSuite], and [RunOptionsSuite]. Plugin authors
// invoke whichever apply.
//
// Failures surface through `t.Errorf` rather than `t.Fatalf` so
// a single failing contract surfaces every downstream failure
// too — the author sees the full report in one run instead of
// chasing one cascade at a time.
func RunSuite(t *testing.T, p plugin.Plugin) {
	t.Helper()
	t.Run("Name returns a non-empty, stable identifier", func(t *testing.T) {
		assertStableName(t, p)
	})
	t.Run("implements at least one of the documented role interfaces", func(t *testing.T) {
		assertImplementsARole(t, p)
	})
	t.Run("CapabilityProvider returns deterministic Provides + Requires", func(t *testing.T) {
		assertCapabilityProviderStability(t, p)
	})
	t.Run("DirectiveProvider schemas declare unique non-empty names", func(t *testing.T) {
		assertDirectiveSchemaUniqueness(t, p)
	})
	t.Run("Versioned reports a stable version string", func(t *testing.T) {
		assertVersionedStability(t, p)
	})
	t.Run("EmitVersioned declares stable, non-empty majors", func(t *testing.T) {
		assertEmitVersionedStability(t, p)
	})
	t.Run("NodesOnly returns a stable declaration", func(t *testing.T) {
		assertNodesOnlyStability(t, p)
	})
	t.Run("FilenameProvider returns stable suffixes per language", func(t *testing.T) {
		assertFilenameProviderStability(t, p)
	})
}

// assertStableName pins Name's empty-string and stability
// contracts: every framework caller treats the result as the
// plugin's identifier — diagnostic attribution, cache-key
// composition, manifest entries all use it.
func assertStableName(tb testing.TB, p plugin.Plugin) {
	tb.Helper()
	first := p.Name()
	if first == "" {
		tb.Errorf("Plugin.Name returned the empty string; framework callers require a stable identifier")
	}
	second := p.Name()
	if first != second {
		tb.Errorf("Plugin.Name not stable across calls: first=%q second=%q", first, second)
	}
}

// assertImplementsARole verifies p satisfies at least one of the
// four role interfaces. A plugin that satisfies none is
// effectively dead code — the pipeline never invokes anything
// on it. The check surfaces this as a contract failure rather
// than a silent no-op at pipeline-Build time.
func assertImplementsARole(tb testing.TB, p plugin.Plugin) {
	tb.Helper()
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
	tb.Errorf(
		"plugin %T implements no role interface "+
			"(Frontend / Annotator / Generator / Backend); pipeline would never invoke it",
		p,
	)
}

// assertCapabilityProviderStability pins the deterministic-
// ordering contract: the resolver depends on Provides / Requires
// returning the same sequence across calls so capability-topo
// ordering stays reproducible.
func assertCapabilityProviderStability(tb testing.TB, p plugin.Plugin) {
	tb.Helper()
	provider, ok := any(p).(plugin.CapabilityProvider)
	if !ok {
		return
	}
	first := provider.Provides()
	second := provider.Provides()
	if !slices.Equal(first, second) {
		tb.Errorf("CapabilityProvider.Provides not stable: first=%v second=%v", first, second)
	}
	firstReq := provider.Requires()
	secondReq := provider.Requires()
	if !slices.Equal(firstReq, secondReq) {
		tb.Errorf("CapabilityProvider.Requires not stable: first=%v second=%v", firstReq, secondReq)
	}
	for _, cap := range first {
		if cap == "" {
			tb.Errorf("CapabilityProvider.Provides contains the empty string; capability labels must be non-empty")
		}
	}
	for _, cap := range firstReq {
		if cap == "" {
			tb.Errorf("CapabilityProvider.Requires contains the empty string; capability labels must be non-empty")
		}
	}
}

// assertDirectiveSchemaUniqueness pins the directive-schema
// contract: every declared schema has a non-empty Name, and no
// two schemas in the same plugin share a Name. Duplicates would
// shadow each other at registration time; the framework's
// directive registry rejects them.
func assertDirectiveSchemaUniqueness(tb testing.TB, p plugin.Plugin) {
	tb.Helper()
	provider, ok := any(p).(plugin.DirectiveProvider)
	if !ok {
		return
	}
	seen := map[directive.Name]struct{}{}
	for _, schema := range provider.Directives() {
		if schema.Name == "" {
			tb.Errorf("DirectiveProvider.Directives contains a schema with empty Name")
			continue
		}
		if _, dup := seen[schema.Name]; dup {
			tb.Errorf("DirectiveProvider.Directives declares duplicate schema name %q", schema.Name)
			continue
		}
		seen[schema.Name] = struct{}{}
	}
}

// assertVersionedStability pins the Versioned contract: when a
// plugin opts into cache invalidation via [plugin.Versioned],
// the returned string must be stable across calls so cache
// keys compose deterministically. The empty string is
// permitted and signals "do not contribute to the cache key";
// the framework's cache machinery treats it as opt-out and
// callers reading the value for header rendering already
// branch on emptiness.
func assertVersionedStability(tb testing.TB, p plugin.Plugin) {
	tb.Helper()
	versioned, ok := any(p).(plugin.Versioned)
	if !ok {
		return
	}
	first := versioned.Version()
	second := versioned.Version()
	if first != second {
		tb.Errorf("Versioned.Version not stable across calls: first=%q second=%q", first, second)
	}
}

// assertEmitVersionedStability pins the EmitVersioned contract:
// the declared major-version list is stable across calls and
// every entry is non-empty so the pipeline's compatibility
// check at Build time produces deterministic results. A plugin
// that declared a list and then mutated it between calls would
// admit or reject the run depending on call ordering — the
// stability check forecloses that surprise.
func assertEmitVersionedStability(tb testing.TB, p plugin.Plugin) {
	tb.Helper()
	ev, ok := any(p).(plugin.EmitVersioned)
	if !ok {
		return
	}
	first := ev.EmitVersions()
	second := ev.EmitVersions()
	if !slices.Equal(first, second) {
		tb.Errorf("EmitVersioned.EmitVersions not stable: first=%v second=%v", first, second)
	}
	for _, v := range first {
		if v == "" {
			tb.Errorf("EmitVersioned.EmitVersions contains the empty string; majors must be non-empty")
		}
	}
}

// assertNodesOnlyStability pins the [plugin.NodesOnly] contract:
// the declaration is static (the docblock calls it a "static
// contract, not a runtime switch") so the pipeline can plan the
// generator phase's parallelisation at Build time. A plugin
// whose NodesOnly toggles between calls would invalidate the
// pipeline's scheduling decision.
func assertNodesOnlyStability(tb testing.TB, p plugin.Plugin) {
	tb.Helper()
	no, ok := any(p).(plugin.NodesOnly)
	if !ok {
		return
	}
	first := no.NodesOnly()
	second := no.NodesOnly()
	if first != second {
		tb.Errorf("NodesOnly not stable across calls: first=%v second=%v", first, second)
	}
}

// assertFilenameProviderStability pins the
// [plugin.FilenameProvider] contract: FilenameSuffix returns
// the same value for the same language across calls. A plugin
// whose suffix flipped between calls would produce different
// filenames on consecutive runs — a layout-determinism
// violation the pipeline cannot recover from.
//
// The check exercises the languages every framework backend
// the suite anticipates encountering uses, plus an empty
// language to verify the plugin doesn't panic on the empty
// string. Plugins that target a language not in this list are
// covered by the per-language stability invariant: the second
// call with the same argument equals the first.
func assertFilenameProviderStability(tb testing.TB, p plugin.Plugin) {
	tb.Helper()
	fp, ok := any(p).(plugin.FilenameProvider)
	if !ok {
		return
	}
	for _, lang := range []string{"go", "rust", "ts", "py", ""} {
		first := fp.FilenameSuffix(lang)
		second := fp.FilenameSuffix(lang)
		if first != second {
			tb.Errorf(
				"FilenameProvider.FilenameSuffix(%q) not stable: first=%q second=%q",
				lang, first, second,
			)
		}
	}
}
