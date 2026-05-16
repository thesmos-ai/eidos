// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/plugin"
)

// TestRunSuite_PassesForWellFormedPlugin pins the happy path of
// the framework conformance suite: the canonical
// [plugintest.FixturePlugin] reference cleanly clears every
// check. The test doubles as a meta-test for the package: any
// suite refactor that breaks the contract surface trips this
// before downstream plugin authors notice.
func TestRunSuite_PassesForWellFormedPlugin(t *testing.T) {
	t.Parallel()

	t.Run("FixturePlugin clears every check", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, plugintest.NewFixturePlugin())
	})

	t.Run("FixturePlugin with empty Version clears the suite", func(t *testing.T) {
		t.Parallel()
		// Empty version is permitted per the [plugin.Versioned]
		// docblock — the plugin opts out of cache integration.
		// The suite must not reject this case.
		p := plugintest.NewFixturePlugin()
		p.VersionString = ""
		plugintest.RunSuite(t, p)
	})

	t.Run("multi-output FixturePlugin clears the suite", func(t *testing.T) {
		t.Parallel()
		// The conformance suite must accept a plugin declaring an
		// ordered set of outputs (primary + tagged secondary) so
		// long as the slice is well-formed: every Suffix
		// non-empty, tags unique, the empty-tag entry at index 0
		// when present.
		plugintest.RunSuite(t, plugintest.NewMultiOutputFixturePlugin())
	})

	t.Run("plugin with every-output-tagged Outputs clears the suite", func(t *testing.T) {
		t.Parallel()
		// A plugin can also declare every output with a non-empty
		// tag (the "no default output" mode). The shape rules
		// permit this — at most one empty-tag output, not
		// exactly one.
		p := plugintest.NewMultiOutputFixturePlugin()
		p.OutputsByLang = map[string][]plugin.Output{
			"go": {
				{Tag: "production", Suffix: "_fixture.go"},
				{Tag: "test", Suffix: "_fixture_test.go"},
			},
		}
		plugintest.RunSuite(t, p)
	})
}

// TestAssertStableName_RejectionPaths covers the two failure
// modes of [plugintest.AssertStableName]: empty identifier and
// mismatched values across calls. Both surface as recorded
// errors on the fake TB.
func TestAssertStableName_RejectionPaths(t *testing.T) {
	t.Parallel()

	t.Run("empty Name is rejected", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		p := plugintest.NewFixturePlugin()
		p.PluginName = ""
		plugintest.AssertStableName(fake, p)
		assertFakeMentions(t, fake, "Plugin.Name returned the empty string")
	})

	t.Run("flapping Name is rejected", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertStableName(fake, &flappingNamePlugin{})
		assertFakeMentions(t, fake, "not stable across calls")
	})
}

// TestAssertImplementsARole_RejectsRoleless covers the role
// probe's only failure mode: a plugin satisfying [plugin.Plugin]
// alone with no role interface. [plugintest.MinimalPlugin] is
// the canonical example.
func TestAssertImplementsARole_RejectsRoleless(t *testing.T) {
	t.Parallel()
	fake := newFakeT()
	plugintest.AssertImplementsARole(fake, plugintest.NewMinimalPlugin("no-role"))
	assertFakeMentions(t, fake, "implements no role interface")
}

// TestAssertCapabilityProviderStability_RejectionPaths pins the
// rejection paths of the capability stability check: empty
// entries in Provides, empty entries in Requires, and lists
// flapping across calls.
func TestAssertCapabilityProviderStability_RejectionPaths(t *testing.T) {
	t.Parallel()

	t.Run("empty Provides entry is rejected", func(t *testing.T) {
		t.Parallel()
		p := plugintest.NewFixturePlugin()
		p.CapabilityProvides = []string{"cap.one", ""}
		fake := newFakeT()
		plugintest.AssertCapabilityProviderStability(fake, p)
		assertFakeMentions(t, fake, "CapabilityProvider.Provides contains the empty string")
	})

	t.Run("empty Requires entry is rejected", func(t *testing.T) {
		t.Parallel()
		p := plugintest.NewFixturePlugin()
		p.CapabilityRequires = []string{""}
		fake := newFakeT()
		plugintest.AssertCapabilityProviderStability(fake, p)
		assertFakeMentions(t, fake, "CapabilityProvider.Requires contains the empty string")
	})

	t.Run("flapping Provides is rejected", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertCapabilityProviderStability(fake, &flappingProvidesPlugin{})
		assertFakeMentions(t, fake, "CapabilityProvider.Provides not stable")
	})
}

// TestAssertDirectiveSchemaUniqueness_RejectionPaths covers the
// empty-name and duplicate-name failure modes.
func TestAssertDirectiveSchemaUniqueness_RejectionPaths(t *testing.T) {
	t.Parallel()

	t.Run("duplicate schema name is rejected", func(t *testing.T) {
		t.Parallel()
		p := plugintest.NewFixturePlugin()
		p.DirectiveSchemas = []directive.Schema{
			directive.NewSchema("foo").On("Struct").Build(),
			directive.NewSchema("foo").On("Interface").Build(),
		}
		fake := newFakeT()
		plugintest.AssertDirectiveSchemaUniqueness(fake, p)
		assertFakeMentions(t, fake, "duplicate schema name")
	})

	t.Run("empty schema name is rejected", func(t *testing.T) {
		t.Parallel()
		p := plugintest.NewFixturePlugin()
		p.DirectiveSchemas = []directive.Schema{{Name: ""}}
		fake := newFakeT()
		plugintest.AssertDirectiveSchemaUniqueness(fake, p)
		assertFakeMentions(t, fake, "empty Name")
	})
}

// TestAssertVersionedStability_RejectsFlapping covers the
// stability rejection for [plugin.Versioned]. Empty is permitted
// and intentionally not a failure mode.
func TestAssertVersionedStability_RejectsFlapping(t *testing.T) {
	t.Parallel()
	fake := newFakeT()
	plugintest.AssertVersionedStability(fake, &flappingVersionPlugin{})
	assertFakeMentions(t, fake, "Versioned.Version not stable")
}

// TestAssertEmitVersionedStability_RejectionPaths covers the
// stability and non-empty-entry rejection paths.
func TestAssertEmitVersionedStability_RejectionPaths(t *testing.T) {
	t.Parallel()

	t.Run("empty entry is rejected", func(t *testing.T) {
		t.Parallel()
		p := plugintest.NewFixturePlugin()
		p.EmitMajors = []string{"1", ""}
		fake := newFakeT()
		plugintest.AssertEmitVersionedStability(fake, p)
		assertFakeMentions(t, fake, "EmitVersioned.EmitVersions contains the empty string")
	})

	t.Run("flapping list is rejected", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertEmitVersionedStability(fake, &flappingEmitVersionsPlugin{})
		assertFakeMentions(t, fake, "EmitVersioned.EmitVersions not stable")
	})
}

// TestAssertNodesOnlyStability_RejectsFlapping covers the
// only failure mode: the declaration flipping between calls.
func TestAssertNodesOnlyStability_RejectsFlapping(t *testing.T) {
	t.Parallel()
	fake := newFakeT()
	plugintest.AssertNodesOnlyStability(fake, &flappingNodesOnlyPlugin{})
	assertFakeMentions(t, fake, "NodesOnly not stable")
}

// TestAssertFilenameProviderStability_RejectsFlapping covers
// the only failure mode of the filename-suffix check.
func TestAssertFilenameProviderStability_RejectsFlapping(t *testing.T) {
	t.Parallel()
	fake := newFakeT()
	plugintest.AssertFilenameProviderStability(fake, &flappingSuffixPlugin{})
	assertFakeMentions(t, fake, "FilenameProvider.Outputs")
}

// TestAssertOutputsShape covers each Outputs-shape rule the
// conformance check enforces — every failure mode reports the
// matching diagnostic; valid slices report nothing.
func TestAssertOutputsShape(t *testing.T) {
	t.Parallel()

	t.Run("rejects an output with empty Suffix", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertOutputsShape(fake, &malformedOutputsPlugin{
			outputs: []plugin.Output{{Suffix: ""}},
		})
		assertFakeMentions(t, fake, "Suffix is required")
	})

	t.Run("rejects duplicate Tag values", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertOutputsShape(fake, &malformedOutputsPlugin{
			outputs: []plugin.Output{
				{Suffix: "_x.go"},
				{Tag: "test", Suffix: "_x_test.go"},
				{Tag: "test", Suffix: "_y_test.go"},
			},
		})
		assertFakeMentions(t, fake, `tag "test" appears at index`)
	})

	t.Run("rejects more than one output with empty Tag", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertOutputsShape(fake, &malformedOutputsPlugin{
			outputs: []plugin.Output{
				{Suffix: "_x.go"},
				{Suffix: "_y.go"},
			},
		})
		assertFakeMentions(t, fake, "outputs declare an empty Tag")
	})

	t.Run("rejects empty-Tag output not at index 0", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertOutputsShape(fake, &malformedOutputsPlugin{
			outputs: []plugin.Output{
				{Tag: "test", Suffix: "_x_test.go"},
				{Suffix: "_x.go"},
			},
		})
		assertFakeMentions(t, fake, "empty-tag output must be declared at index 0")
	})

	t.Run("accepts a well-formed multi-output slice silently", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertOutputsShape(fake, &malformedOutputsPlugin{
			outputs: []plugin.Output{
				{Suffix: "_x.go"},
				{Tag: "test", Suffix: "_x_test.go"},
			},
		})
		if fake.failed {
			t.Fatalf("well-formed Outputs reported a failure: errs=%v fatals=%v", fake.errs, fake.fatals)
		}
	})

	t.Run("non-implementer passes silently", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		plugintest.AssertOutputsShape(fake, plugintest.NewFixturePlugin())
		// FixturePlugin implements FilenameProvider with a valid
		// single-output slice for "go" only; every other language
		// returns nil and is therefore skipped.
		if fake.failed {
			t.Fatalf("default fixture reported a failure: errs=%v fatals=%v", fake.errs, fake.fatals)
		}
	})
}

// assertFakeMentions fails t when none of the recorded error /
// fatal strings in fake contain substr.
func assertFakeMentions(t *testing.T, fake *fakeT, substr string) {
	t.Helper()
	joined := strings.Join(append(fake.errs, fake.fatals...), "\n")
	if !strings.Contains(joined, substr) {
		t.Errorf("fake TB did not record a message mentioning %q; got:\n%s", substr, joined)
	}
}
