// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
)

// assertEqualString fails the test if got and want differ.
func assertEqualString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string mismatch:\n got:  %q\n want: %q", got, want)
	}
}

// directiveAt builds a directive instance with a name and position.
// Used across tests that need to populate BaseNode.DirectiveList.
func directiveAt(name directive.Name, pos position.Pos) *directive.Directive {
	return &directive.Directive{Name: name, Pos: pos, KV: map[string]string{}}
}

// namedRef builds a Named TypeRef. Used by tests that need a quick
// reference to a type without full backstory.
func namedRef(pkg, name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Package: pkg, Name: name}
}

// constraintFrom builds a [node.Constraint] embedding refs as named
// bounds. Used by tests that need a quick generic-constraint instance
// without manual struct-literal noise.
func constraintFrom(refs ...*node.TypeRef) *node.Constraint {
	return &node.Constraint{Embedded: refs}
}

// recordingVisitor collects the [directive.Kind] of every node Walk
// visits, in visit order. Tests assert on the resulting slice.
type recordingVisitor struct {
	kinds *[]directive.Kind
}

// Visit appends n.Kind() to the underlying slice and continues
// descent.
func (r recordingVisitor) Visit(n node.Node) node.Visitor {
	*r.kinds = append(*r.kinds, n.Kind())
	return r
}

// recordWalk runs [node.Walk] starting from n and returns the
// [directive.Kind] of every visited node in visit order.
func recordWalk(n node.Node) []directive.Kind {
	var kinds []directive.Kind
	node.Walk(n, recordingVisitor{kinds: &kinds})
	return kinds
}

// byteSliceRef returns a slice-of-byte [node.TypeRef], a common
// helper across anonymous-interface and signature-comparison tests.
func byteSliceRef() *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: namedRef("", "byte")}
}

// anonInterfaceWithMethod returns a [node.TypeRefAnonInterface] ref
// carrying a single method with the supplied shape.
func anonInterfaceWithMethod(
	name string,
	params []*node.Param,
	returns []*node.TypeRef,
	variadic bool,
) *node.TypeRef {
	if variadic && len(params) > 0 {
		params[len(params)-1].Variadic = true
	}
	return &node.TypeRef{
		TypeKind: node.TypeRefAnonInterface,
		Methods:  []*node.Method{{Name: name, Params: params, Returns: returns}},
	}
}

// anonInterfaceWithRead is an [anonInterfaceWithMethod] shorthand for
// the canonical "Read" method shape used across anon-interface
// equality tests.
func anonInterfaceWithRead(
	params []*node.Param,
	returns []*node.TypeRef,
	variadic bool,
) *node.TypeRef {
	return anonInterfaceWithMethod("Read", params, returns, variadic)
}
