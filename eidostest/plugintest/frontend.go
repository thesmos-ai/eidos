// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

// FrontendFixture describes a single source-loading scenario
// the [RunFrontendSuite] drives a [plugin.Frontend] against.
// The fixture declares the user-facing Pattern and the
// per-plugin Options the frontend's [plugin.OptionsProvider]
// (when implemented) decodes via SetOptions.
//
// Frontends typically encode their input root through Options
// (`dir`, `root`, `entrypoint`, …) rather than through the
// pattern — patterns scope what to load within that root. The
// suite calls SetOptions once with the supplied map before each
// Load invocation, so the frontend re-receives its configured
// values across the two determinism passes.
type FrontendFixture struct {
	// Name labels the fixture in subtest paths and failure
	// messages. Required and unique within a single
	// [RunFrontendSuite] call.
	Name string

	// Pattern is the literal value supplied to
	// [plugin.FrontendContext.Pattern]. Typically "./..." or a
	// language-specific glob; the frontend interprets it
	// language-appropriately.
	Pattern string

	// Options is forwarded verbatim to the frontend's
	// [plugin.OptionsProvider.SetOptions] when the frontend
	// implements that capability. Frontends that do not
	// implement OptionsProvider ignore this map; the suite
	// surfaces a contract failure if the frontend implements the
	// interface but rejects the supplied values.
	Options map[string]string
}

// RunFrontendSuite runs the conformance checks every
// [plugin.Frontend] must satisfy: Load must not panic on an
// empty / minimal context; for each fixture, Load must succeed
// without panicking; and Load must be deterministic — two
// invocations driven against equivalent contexts (independent
// stores, same pattern, same options) must produce equivalent
// node graphs.
//
// The suite calls [plugin.OptionsProvider.SetOptions] before
// every Load when the frontend implements the capability, so
// the fixture's Options apply both to the panic-recovery probe
// and to the determinism passes. Pass an empty fixture slice to
// run only the empty-pattern contract.
func RunFrontendSuite(t *testing.T, f plugin.Frontend, fixtures []FrontendFixture) {
	t.Helper()
	t.Run("Load on empty pattern does not panic", func(t *testing.T) {
		assertLoadEmptyPatternDoesNotPanic(t, f)
	})
	assertFrontendFixtureNamesUnique(t, fixtures)
	for _, fx := range fixtures {
		t.Run("fixture="+fx.Name+"/Load does not panic", func(t *testing.T) {
			assertLoadDoesNotPanic(t, f, fx)
		})
		t.Run("fixture="+fx.Name+"/Load is deterministic across two runs", func(t *testing.T) {
			assertLoadIsDeterministic(t, f, fx)
		})
	}
}

// assertFrontendFixtureNamesUnique fails when two fixtures
// share a Name.
func assertFrontendFixtureNamesUnique(tb testing.TB, fixtures []FrontendFixture) {
	tb.Helper()
	seen := make(map[string]struct{}, len(fixtures))
	for _, fx := range fixtures {
		if fx.Name == "" {
			tb.Fatalf("RunFrontendSuite: fixture has empty Name; every FrontendFixture must declare one")
		}
		if _, dup := seen[fx.Name]; dup {
			tb.Fatalf("RunFrontendSuite: duplicate fixture Name %q", fx.Name)
		}
		seen[fx.Name] = struct{}{}
	}
}

// assertLoadEmptyPatternDoesNotPanic drives the frontend with
// an empty pattern and an otherwise minimal context. The
// frontend is permitted to fail (returning a non-nil error or
// surfacing diagnostics) on an empty pattern — the contract
// here is the narrower no-panic invariant. Panics on an empty
// pattern crash the process on projects whose patterns expand
// to nothing.
func assertLoadEmptyPatternDoesNotPanic(tb testing.TB, f plugin.Frontend) {
	tb.Helper()
	s := store.New()
	if err := runLoadRecovering(f, FrontendFixture{}, s); err != nil {
		if strings.HasPrefix(err.Error(), "recovered panic") {
			tb.Errorf("Load panicked on empty pattern: %v", err)
		}
	}
}

// assertLoadDoesNotPanic drives the frontend against the
// fixture's pattern and fails if Load panics. Returned errors
// are not a contract failure on their own — frontends surface
// per-input issues through ctx.Diag and reserve the return
// value for catastrophic failures the suite can't classify.
func assertLoadDoesNotPanic(tb testing.TB, f plugin.Frontend, fx FrontendFixture) {
	tb.Helper()
	s := store.New()
	err := runLoadRecovering(f, fx, s)
	if err != nil && strings.HasPrefix(err.Error(), "recovered panic") {
		tb.Errorf("Load panicked on fixture %q: %v", fx.Name, err)
	}
}

// assertLoadIsDeterministic drives Load twice against fresh
// stores and compares the resulting node-graph projections.
// The projection is a sorted slice of stable identity tuples
// covering every indexed node — kind, qualified name, package
// — so the diff catches missing or extra nodes the frontend
// produced inconsistently across runs.
//
// Per-node detail (positions, doc comments, directive args) is
// outside the determinism check's scope: downstream tests
// assert against full source mapping through [frontendtest],
// where divergences surface with line-level context. The
// projection here pins the structural-determinism property the
// pipeline's cache and incremental-rebuild paths rely on.
func assertLoadIsDeterministic(tb testing.TB, f plugin.Frontend, fx FrontendFixture) {
	tb.Helper()
	first := store.New()
	if err := runLoadRecovering(f, fx, first); err != nil {
		if strings.HasPrefix(err.Error(), "recovered panic") {
			tb.Fatalf("Load panicked on first determinism pass of fixture %q: %v", fx.Name, err)
		}
		// A return error from Load is permitted (the frontend
		// may surface contract failures through it). The
		// determinism check still runs against the partial
		// store the frontend populated; abort only on panic.
	}
	second := store.New()
	if err := runLoadRecovering(f, fx, second); err != nil {
		if strings.HasPrefix(err.Error(), "recovered panic") {
			tb.Fatalf("Load panicked on second determinism pass of fixture %q: %v", fx.Name, err)
		}
	}
	firstProj := nodeProjection(first)
	secondProj := nodeProjection(second)
	if !slices.Equal(firstProj, secondProj) {
		tb.Errorf(
			"node projection differs across two runs of fixture %q; frontend is not deterministic\n"+
				"  first run:  %s\n  second run: %s",
			fx.Name,
			strings.Join(firstProj, ", "), strings.Join(secondProj, ", "),
		)
	}
}

// runLoadRecovering applies fixture options (when the frontend
// supports OptionsProvider) and invokes Load with a discard
// diagnostic sink. Panics are recovered into a returned error
// prefixed with "recovered panic"; clean Load returns are
// passed through unmodified.
func runLoadRecovering(f plugin.Frontend, fx FrontendFixture, s *store.Store) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered panic: %v", r)
		}
	}()
	if provider, ok := any(f).(plugin.OptionsProvider); ok {
		if serr := provider.SetOptions(opt.New(provider.OptionsSchema(), fx.Options)); serr != nil {
			return fmt.Errorf("set options: %w", serr)
		}
	}
	ctx := &plugin.FrontendContext{
		Store:    s,
		Diag:     diag.New(),
		Registry: directive.NewRegistry(),
		Parser:   directive.DefaultParser(),
		Cache:    cache.NewNone(),
		Pattern:  fx.Pattern,
	}
	if rerr := f.Load(ctx); rerr != nil {
		return fmt.Errorf("load returned error: %w", rerr)
	}
	return nil
}

// nodeOwnerName returns the qualified name of a source-node
// owner pointer (Method.Owner / Field.Owner). The owner is
// always a kind that implements QName when set; nil owners
// surface as [unownedSentinel] so failure output stays readable.
func nodeOwnerName(owner any) string {
	if owner == nil {
		return unownedSentinel
	}
	if q, ok := owner.(interface{ QName() string }); ok {
		return q.QName()
	}
	return unnamedSentinel
}

// nodeProjection returns a sorted slice of stable identity
// strings — one per indexed node in s — covering every kind the
// suite recognises. Mirrors [emitProjection] in shape; the
// frontend suite uses it for determinism comparison.
func nodeProjection(s *store.Store) []string {
	nv := s.Nodes()
	total := nv.Packages().Len() + nv.Files().Len() + nv.Imports().Len() +
		nv.Structs().Len() + nv.Interfaces().Len() + nv.Methods().Len() +
		nv.Fields().Len() + nv.Functions().Len() + nv.Variables().Len() +
		nv.Constants().Len() + nv.Enums().Len() + nv.EnumVariants().Len() +
		nv.Aliases().Len()
	out := make([]string, 0, total)
	for _, n := range nv.Packages().Items() {
		out = append(out, fmt.Sprintf("package:%s:%s", n.Name, n.Path))
	}
	for _, n := range nv.Files().Items() {
		out = append(out, "file:"+n.Path)
	}
	for _, n := range nv.Imports().Items() {
		out = append(out, fmt.Sprintf("import:%s:alias=%s", n.Path, n.Alias))
	}
	for _, n := range nv.Structs().Items() {
		out = append(out, "struct:"+n.QName())
	}
	for _, n := range nv.Interfaces().Items() {
		out = append(out, "interface:"+n.QName())
	}
	for _, n := range nv.Methods().Items() {
		out = append(out, fmt.Sprintf("method:%s.%s", nodeOwnerName(n.Owner), n.Name))
	}
	for _, n := range nv.Fields().Items() {
		out = append(out, fmt.Sprintf("field:%s.%s", nodeOwnerName(n.Owner), n.Name))
	}
	for _, n := range nv.Functions().Items() {
		out = append(out, "function:"+n.QName())
	}
	for _, n := range nv.Variables().Items() {
		out = append(out, "variable:"+n.QName())
	}
	for _, n := range nv.Constants().Items() {
		out = append(out, "constant:"+n.QName())
	}
	for _, n := range nv.Enums().Items() {
		out = append(out, "enum:"+n.QName())
	}
	for _, n := range nv.EnumVariants().Items() {
		owner := unownedSentinel
		if n.Owner != nil {
			owner = n.Owner.QName()
		}
		out = append(out, fmt.Sprintf("enum-variant:%s.%s", owner, n.Name))
	}
	for _, n := range nv.Aliases().Items() {
		out = append(out, "alias:"+n.QName())
	}
	slices.Sort(out)
	return out
}
