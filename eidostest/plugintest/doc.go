// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package plugintest ships the framework-conformance suite plugin
// authors run against their plugin instances. The suite is split
// into a universal framework check ([RunSuite]) and per-role
// contract checks the plugin author invokes for whichever roles
// the plugin implements ([RunAnnotatorSuite], [RunGeneratorSuite],
// [RunBackendSuite], [RunFrontendSuite]) plus an options
// round-trip suite ([RunOptionsSuite]) for plugins implementing
// [plugin.OptionsProvider].
//
// # Universal contracts
//
// [RunSuite] is appropriate for every plugin and pins the
// invariants the pipeline relies on at registration / build time:
// stable [plugin.Plugin.Name], at-least-one role-interface
// compliance, deterministic [plugin.CapabilityProvider] ordering,
// unique [plugin.DirectiveProvider] schema names, well-formed
// [plugin.EmitVersioned] entries, stable [plugin.NodesOnly]
// declaration, and stable [plugin.FilenameProvider.FilenameSuffix]
// per language. Each contract is its own subtest so a single
// regression surfaces with the specific contract that broke.
//
// # Per-role contracts
//
// Per-role suites accept a per-role fixture describing the input
// the suite drives the plugin against. The fixtures are the
// surface the plugin author tailors to the plugin's domain —
// e.g. an annotator's [AnnotatorFixture] supplies a store builder
// that lays down whatever source-side decls the annotator's
// stamping logic needs. The suite drives the plugin against the
// fixture twice (or against two equivalent inputs, for the cases
// where re-running on the same state is not meaningful) and
// asserts the two passes produce identical output — the
// determinism contract every pipeline phase relies on for
// reproducible builds, byte-stable goldens, and cache
// composability.
//
// # Diagnostic discipline
//
// Each per-role suite verifies the plugin's diagnostic
// discipline: malformed inputs and edge-case stores surface
// as positioned diagnostics, never as panics or arbitrary error
// returns. A plugin that panics on an empty store or a fixture
// it cannot handle is a contract failure the pipeline cannot
// recover from at runtime.
//
// # Typical usage
//
//	func TestMyPlugin(t *testing.T) {
//	    p := myplugin.New()
//	    plugintest.RunSuite(t, p)
//	    plugintest.RunGeneratorSuite(t, p, []plugintest.GeneratorFixture{{
//	        Name: "single struct with +gen:my",
//	        BuildStore: func(t *testing.T) *store.Store {
//	            return storefixture.New().
//	                Struct("User", func(s *storefixture.StructBuilder) {
//	                    s.Directive(storefixture.Directive("my"))
//	                }).Build()
//	        },
//	    }})
//	    plugintest.RunOptionsSuite(t, p, plugintest.OptionsFixture{
//	        Valid: map[string]string{"output_package": "main"},
//	        UnknownKey: "no_such_key",
//	    })
//	}
//
// Plugin authors invoke whichever suites apply; the framework
// emits no implicit calls. Subtests scope the suite's checks so
// `go test -run` patterns isolate a single contract.
package plugintest
