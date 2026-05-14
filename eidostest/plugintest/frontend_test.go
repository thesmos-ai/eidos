// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest_test

import (
	"errors"
	"fmt"
	"testing"

	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
)

// TestRunFrontendSuite_PassesForWellFormedFrontend pins the
// happy path: a frontend that emits a stable per-pattern node
// graph passes every contract.
func TestRunFrontendSuite_PassesForWellFormedFrontend(t *testing.T) {
	t.Parallel()
	plugintest.RunFrontendSuite(
		t,
		&fakeFrontend{name: "fake-fe"},
		[]plugintest.FrontendFixture{
			{
				Name:    "single-package pattern",
				Pattern: "single",
				Options: map[string]string{"label": "alpha"},
			},
			{
				Name:    "two-package pattern",
				Pattern: "two",
				Options: map[string]string{"label": "beta"},
			},
		},
	)
}

// TestRunFrontendSuite_RejectsPanickingFrontend covers the
// empty-pattern panic rejection.
func TestRunFrontendSuite_RejectsPanickingFrontend(t *testing.T) {
	t.Parallel()
	f := &panickingFrontend{name: "panicky"}
	fake := newFakeT()
	plugintest.AssertLoadEmptyPatternDoesNotPanic(fake, f)
	assertFakeMentions(t, fake, "Load panicked on empty pattern")
}

// TestRunFrontendSuite_RejectsNonDeterministicFrontend pins
// the determinism contract: a frontend whose output varies
// across calls (per-call counter, time-derived names) fails
// the comparison.
func TestRunFrontendSuite_RejectsNonDeterministicFrontend(t *testing.T) {
	t.Parallel()
	f := &flappingFrontend{name: "flap"}
	fx := plugintest.FrontendFixture{Name: "single", Pattern: "single"}
	fake := newFakeT()
	plugintest.AssertLoadIsDeterministic(fake, f, fx)
	assertFakeMentions(t, fake, "frontend is not deterministic")
}

// TestRunFrontendSuite_FailsOnDuplicateFixtureName pins the
// fixture-name uniqueness contract.
func TestRunFrontendSuite_FailsOnDuplicateFixtureName(t *testing.T) {
	t.Parallel()
	fixtures := []plugintest.FrontendFixture{
		{Name: "dup", Pattern: "p"},
		{Name: "dup", Pattern: "p"},
	}
	fake := newFakeT()
	captureFatal(func() {
		plugintest.AssertFrontendFixtureNamesUnique(fake, fixtures)
	})
	assertFakeMentions(t, fake, "duplicate fixture Name")
}

// fakeFrontend is an in-memory frontend that populates the
// store from a pattern-keyed lookup table. Deterministic by
// construction — equivalent patterns produce equivalent node
// graphs.
type fakeFrontend struct {
	name string
	opts fakeFrontendOpts
}

// fakeFrontendOpts is the bound options the fake frontend
// exposes through OptionsProvider so the suite exercises that
// integration path.
type fakeFrontendOpts struct {
	// Label is a free-text annotation the suite passes through
	// fixtures; the frontend stores it on receiver state.
	Label string `eidos:"label"`
}

// Name returns the configured identifier.
func (f *fakeFrontend) Name() string { return f.name }

// OptionsSchema returns the reflected schema of
// [fakeFrontendOpts].
func (*fakeFrontend) OptionsSchema() opt.Schema { return opt.Reflect(fakeFrontendOpts{}) }

// SetOptions decodes opts into the frontend's options.
func (f *fakeFrontend) SetOptions(opts opt.Options) error {
	if err := opts.Decode(&f.opts); err != nil {
		return fmt.Errorf("fakeFrontend: SetOptions: %w", err)
	}
	return nil
}

// Load adds packages keyed by the pattern. Unknown patterns
// add nothing; the empty pattern is treated as "no input" and
// returns cleanly.
func (f *fakeFrontend) Load(ctx *plugin.FrontendContext) error {
	switch ctx.Pattern {
	case "single":
		return f.addPackage(ctx, "example.com/one", "Alpha", "Beta")
	case "two":
		if err := f.addPackage(ctx, "example.com/one", "Alpha"); err != nil {
			return err
		}
		return f.addPackage(ctx, "example.com/two", "Gamma")
	default:
		return nil
	}
}

// addPackage seeds a single package with the supplied struct
// names. Helper for the per-pattern dispatch above.
func (*fakeFrontend) addPackage(ctx *plugin.FrontendContext, path string, structNames ...string) error {
	pkg := &node.Package{Name: pkgShortName(path), Path: path}
	for _, name := range structNames {
		pkg.Structs = append(pkg.Structs, &node.Struct{Name: name, Package: path})
	}
	if err := ctx.Store.Nodes().AddPackage(pkg); err != nil {
		return fmt.Errorf("fakeFrontend: AddPackage %q: %w", path, err)
	}
	return nil
}

// pkgShortName extracts a short package name from a slash-
// separated path. Helper for [fakeFrontend.addPackage].
func pkgShortName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// panickingFrontend panics in Load. Used to verify the
// empty-pattern panic-recovery probe.
type panickingFrontend struct{ name string }

// Name returns the configured identifier.
func (f *panickingFrontend) Name() string { return f.name }

// Load panics on every invocation.
func (*panickingFrontend) Load(_ *plugin.FrontendContext) error {
	panic("plugintest test: panickingFrontend panicking on purpose") //nolint:forbidigo
}

// flappingFrontend produces a different node graph on each
// call by embedding a per-instance counter in the struct names.
type flappingFrontend struct {
	name  string
	count int
}

// Name returns the configured identifier.
func (f *flappingFrontend) Name() string { return f.name }

// Load adds one struct whose name embeds the per-call counter.
func (f *flappingFrontend) Load(ctx *plugin.FrontendContext) error {
	f.count++
	pkg := &node.Package{Name: "flap", Path: "example.com/flap"}
	pkg.Structs = []*node.Struct{{
		Name:    fmt.Sprintf("Flap%d", f.count),
		Package: pkg.Path,
	}}
	if err := ctx.Store.Nodes().AddPackage(pkg); err != nil {
		return fmt.Errorf("flappingFrontend: AddPackage: %w", err)
	}
	return errors.New("flappingFrontend: deliberate return error") //nolint:err113
}
