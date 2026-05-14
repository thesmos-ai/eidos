// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest

// Test-only aliases that expose the package's internal
// assertion functions through the black-box `plugintest_test`
// package so the rejection-path tests can drive each check
// individually against a recording fake [testing.TB] — the
// top-level [RunSuite] / [RunAnnotatorSuite] / etc. entry
// points take `*testing.T`, which we cannot fabricate outside
// the testing harness.
var (
	// Framework-level assertions.
	AssertStableName                  = assertStableName
	AssertImplementsARole             = assertImplementsARole
	AssertCapabilityProviderStability = assertCapabilityProviderStability
	AssertDirectiveSchemaUniqueness   = assertDirectiveSchemaUniqueness
	AssertVersionedStability          = assertVersionedStability
	AssertEmitVersionedStability      = assertEmitVersionedStability
	AssertNodesOnlyStability          = assertNodesOnlyStability
	AssertFilenameProviderStability   = assertFilenameProviderStability

	// Annotator-suite assertions.
	AssertAnnotateEmptyStoreDoesNotPanic   = assertAnnotateEmptyStoreDoesNotPanic
	AssertAnnotateDoesNotPanic             = assertAnnotateDoesNotPanic
	AssertAnnotateLeavesNodeCountUnchanged = assertAnnotateLeavesNodeCountUnchanged
	AssertAnnotateIsIdempotent             = assertAnnotateIsIdempotent
	AssertAnnotatorFixtureNamesUnique      = assertAnnotatorFixtureNamesUnique

	// Generator-suite assertions.
	AssertGenerateEmptyStoreDoesNotPanic     = assertGenerateEmptyStoreDoesNotPanic
	AssertGenerateDoesNotPanic               = assertGenerateDoesNotPanic
	AssertGenerateLeavesSourceNodesUnchanged = assertGenerateLeavesSourceNodesUnchanged
	AssertGenerateIsDeterministic            = assertGenerateIsDeterministic
	AssertGeneratorFixtureNamesUnique        = assertGeneratorFixtureNamesUnique

	// Backend-suite assertions.
	AssertRenderEmptyEmitDoesNotPanic = assertRenderEmptyEmitDoesNotPanic
	AssertRenderDoesNotPanic          = assertRenderDoesNotPanic
	AssertRenderCarriesNoErrors       = assertRenderCarriesNoErrors
	AssertRenderIsByteStable          = assertRenderIsByteStable
	AssertBackendFixtureNamesUnique   = assertBackendFixtureNamesUnique

	// Frontend-suite assertions.
	AssertLoadEmptyPatternDoesNotPanic = assertLoadEmptyPatternDoesNotPanic
	AssertLoadDoesNotPanic             = assertLoadDoesNotPanic
	AssertLoadIsDeterministic          = assertLoadIsDeterministic
	AssertFrontendFixtureNamesUnique   = assertFrontendFixtureNamesUnique

	// Options-suite assertions.
	AssertOptionsSchemaStability       = assertOptionsSchemaStability
	AssertOptionsFixtureCoversRequired = assertOptionsFixtureCoversRequired
	AssertSetOptionsAcceptsValid       = assertSetOptionsAcceptsValid
	AssertSetOptionsRejectsUnknown     = assertSetOptionsRejectsUnknown
)
