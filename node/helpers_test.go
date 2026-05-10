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
