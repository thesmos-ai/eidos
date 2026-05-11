// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// assertEqualString fails the test if got and want differ.
func assertEqualString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string mismatch:\n got:  %q\n want: %q", got, want)
	}
}

// assertNoError fails the test if err is non-nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// directiveAt builds a directive instance with a name and position.
func directiveAt(name directive.Name, pos position.Pos) *directive.Directive {
	return &directive.Directive{Name: name, Pos: pos, KV: map[string]string{}}
}

// builtinRef builds a [emit.BuiltinRef] for the named primitive type.
func builtinRef(name string) *emit.BuiltinRef {
	return emit.Builtin(name)
}

// externalRef builds a [emit.ExternalRef] for the named third-party
// type.
func externalRef(pkg, name string) *emit.ExternalRef {
	return emit.External(pkg, name)
}

// constraintFrom builds a [emit.Constraint] embedding refs as named
// bounds. Used by tests that need a quick generic-constraint instance
// without manual struct-literal noise.
func constraintFrom(refs ...emit.Ref) *emit.Constraint {
	return &emit.Constraint{Embedded: refs}
}

// recordingVisitor collects the directive.Kind of every node Walk
// visits, in visit order. Tests assert on the resulting slice.
type recordingVisitor struct {
	kinds *[]directive.Kind
}

// Visit appends the visited node's kind and continues descent.
func (r recordingVisitor) Visit(n emit.Node) emit.Visitor {
	*r.kinds = append(*r.kinds, n.Kind())
	return r
}

// recordWalk runs [emit.Walk] over n and returns the visit-order
// list of node kinds.
func recordWalk(n emit.Node) []directive.Kind {
	var kinds []directive.Kind
	emit.Walk(n, recordingVisitor{kinds: &kinds})
	return kinds
}
