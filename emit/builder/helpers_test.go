// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// defaultTarget is the canonical target every test fixture writes
// into. Centralised so an audit of "where does this fixture emit?"
// reads from one place.
var defaultTarget = emit.Target{Dir: "users", Filename: "users.go", Package: "users"}

// assertCommon asserts the Pos / Docs / Directive trio every
// decl-level builder shares. Used by every per-builder accessor
// test; centralised so the per-builder test stanzas stay focused
// on the bits that vary.
func assertCommon(
	t *testing.T,
	gotPos position.Pos,
	gotDocs []string,
	gotDirs []*directive.Directive,
	wantPos position.Pos,
	wantDir *directive.Directive,
) {
	t.Helper()
	if gotPos != wantPos {
		t.Fatalf("Pos = %v, want %v", gotPos, wantPos)
	}
	if len(gotDocs) != 1 {
		t.Fatalf("expected one doc line; got %+v", gotDocs)
	}
	if len(gotDirs) != 1 || gotDirs[0] != wantDir {
		t.Fatalf("expected the supplied directive in DirectiveList; got %+v", gotDirs)
	}
}

// fixturePos returns the positional fixture every accessor test
// stamps as the source position. Distinct from a zero-value
// position so override detection is unambiguous.
func fixturePos() position.Pos { return position.Pos{File: "fixture.go", Line: 10} }

// fixtureDirective returns a synthetic [*directive.Directive] used
// by every accessor test to verify the Directive accessor threads
// the value through.
func fixtureDirective() *directive.Directive {
	return &directive.Directive{Name: directive.Name("gen:probe")}
}

// otherTarget returns a target distinct from [defaultTarget], used
// by per-builder Target / File override tests.
func otherTarget() emit.Target {
	return emit.Target{Dir: "other", Filename: "other.go", Package: "other"}
}

// fixtureOrigin returns a [node.Node] every per-builder accessor test
// passes to the Origin fluent setter. Using a node.Struct keeps the
// fixture concrete; any node.Node satisfies the interface.
func fixtureOrigin() node.Node {
	return &node.Struct{Name: "Origin"}
}

// fieldNames returns the names of fields contributed to a struct's
// FieldsSlot in insertion order. The cross-cutting slot view (not
// the typed Fields slice) is what cross-cutting plugin inserts
// land in.
func fieldNames(s *emit.Struct) []string {
	slot := s.FieldsSlot()
	out := make([]string, slot.Len())
	for i := range slot.Len() {
		out[i] = slot.At(i).(*emit.Field).Name
	}
	return out
}

// methodNamesFromSlot returns the names of methods in a methods
// slot in insertion order. Used by the cross-cutting method-insert
// tests across Struct / Interface / Alias hosts.
func methodNamesFromSlot(slot *emit.Slot) []string {
	out := make([]string, slot.Len())
	for i := range slot.Len() {
		out[i] = slot.At(i).(*emit.Method).Name
	}
	return out
}
