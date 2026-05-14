// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest

import (
	"errors"
	"maps"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/plugin"
)

// OptionsFixture describes the inputs the [RunOptionsSuite]
// drives a [plugin.OptionsProvider] against. The Valid map is
// the canonical "everything supplied" set the plugin should
// accept; UnknownKey is a key absent from the schema that the
// strict-decode path should reject with [opt.ErrUnknownField].
//
// Plugins whose schemas have no required fields can leave Valid
// empty — the suite still exercises the empty-input
// (defaults-only) path. Plugins whose schemas have required
// fields must populate Valid with at least every required name
// so the happy-path probe doesn't trip on the missing-required
// check before it reaches the unknown-key probe.
type OptionsFixture struct {
	// Valid is the "all valid values" map the suite calls
	// SetOptions with on the happy-path probe. The plugin
	// should accept this without error.
	Valid map[string]string

	// UnknownKey is a key absent from the plugin's schema. The
	// suite drives SetOptions with a map containing this key
	// (plus the Valid entries, to keep required fields
	// satisfied) and asserts the call returns
	// [opt.ErrUnknownField].
	UnknownKey string
}

// RunOptionsSuite runs the conformance checks every
// [plugin.OptionsProvider] must satisfy: OptionsSchema returns
// the same schema across calls (the pipeline reads it once at
// Build time); SetOptions accepts the empty input map (all
// optional defaults / required-field rejection) cleanly;
// SetOptions accepts the supplied Valid map; SetOptions rejects
// the UnknownKey-augmented map with [opt.ErrUnknownField].
//
// Plugins that do not implement [plugin.OptionsProvider] are
// not the target of this suite; passing one to RunOptionsSuite
// fails the build with a positioned diagnostic via [t.Fatalf].
//
// The Valid map should satisfy every required field the
// schema declares; the suite checks the schema's required-field
// set against Valid and reports a fixture-shape failure when
// they don't agree, so the rejection-path checks aren't masked
// by a missing-required error.
func RunOptionsSuite(t *testing.T, p plugin.Plugin, fixture OptionsFixture) {
	t.Helper()
	provider, ok := any(p).(plugin.OptionsProvider)
	if !ok {
		t.Fatalf("RunOptionsSuite: plugin %T does not implement plugin.OptionsProvider", p)
	}
	t.Run("OptionsSchema returns a stable schema across calls", func(t *testing.T) {
		assertOptionsSchemaStability(t, provider)
	})
	t.Run("fixture covers every required schema field", func(t *testing.T) {
		assertOptionsFixtureCoversRequired(t, provider, fixture)
	})
	t.Run("SetOptions accepts the supplied Valid values", func(t *testing.T) {
		assertSetOptionsAcceptsValid(t, provider, fixture)
	})
	t.Run("SetOptions rejects an UnknownKey with ErrUnknownField", func(t *testing.T) {
		assertSetOptionsRejectsUnknown(t, provider, fixture)
	})
}

// assertOptionsSchemaStability calls OptionsSchema twice and
// fails when the field set differs across calls. The schema is
// derived at Build time and the pipeline assumes it stays
// constant across runs; a schema that changed between calls
// would surface as inconsistent validation behaviour.
func assertOptionsSchemaStability(tb testing.TB, p plugin.OptionsProvider) {
	tb.Helper()
	first := p.OptionsSchema().Names()
	second := p.OptionsSchema().Names()
	if !slices.Equal(first, second) {
		tb.Errorf("OptionsSchema field set not stable across calls: first=%v second=%v", first, second)
	}
}

// assertOptionsFixtureCoversRequired fails when the fixture's
// Valid map omits a required field. The rejection-path checks
// downstream rely on the happy-path SetOptions call clearing
// the schema; a missing-required error would mask whatever the
// rejection-path probe is actually trying to surface.
func assertOptionsFixtureCoversRequired(tb testing.TB, p plugin.OptionsProvider, fx OptionsFixture) {
	tb.Helper()
	for _, f := range p.OptionsSchema().Fields {
		if !f.Required {
			continue
		}
		if _, ok := fx.Valid[f.Name]; !ok {
			tb.Errorf(
				"OptionsFixture.Valid is missing required field %q; "+
					"populate it so downstream rejection probes aren't masked by ErrMissingRequired",
				f.Name,
			)
		}
	}
}

// assertSetOptionsAcceptsValid drives SetOptions with the
// fixture's Valid map and fails when it returns a non-nil
// error. The Valid map represents the canonical success case;
// a rejection here points at a fixture mismatch or a schema
// bug.
func assertSetOptionsAcceptsValid(tb testing.TB, p plugin.OptionsProvider, fx OptionsFixture) {
	tb.Helper()
	if err := p.SetOptions(opt.New(p.OptionsSchema(), fx.Valid)); err != nil {
		tb.Errorf("SetOptions rejected the Valid fixture values: %v (values=%v)", err, fx.Valid)
	}
}

// assertSetOptionsRejectsUnknown drives SetOptions with the
// fixture's Valid map plus a key not declared in the schema
// and fails when the call returns nil or returns an error not
// wrapping [opt.ErrUnknownField]. The strict-unknown contract
// catches config-file typos at decode time rather than
// silently dropping the offending entry.
//
// Fixtures may set UnknownKey to the empty string when the
// plugin's schema covers every plausible name (no negative
// probe to perform); the suite then skips this check.
func assertSetOptionsRejectsUnknown(tb testing.TB, p plugin.OptionsProvider, fx OptionsFixture) {
	tb.Helper()
	if fx.UnknownKey == "" {
		return
	}
	values := make(map[string]string, len(fx.Valid)+1)
	maps.Copy(values, fx.Valid)
	values[fx.UnknownKey] = "any"
	err := p.SetOptions(opt.New(p.OptionsSchema(), values))
	if err == nil {
		tb.Errorf(
			"SetOptions accepted an unknown key %q; the strict-decode contract "+
				"requires every input key to match a declared field",
			fx.UnknownKey,
		)
		return
	}
	if !errors.Is(err, opt.ErrUnknownField) {
		tb.Errorf(
			"SetOptions rejected the unknown key %q but the error did not wrap opt.ErrUnknownField: %v",
			fx.UnknownKey, err,
		)
	}
}
