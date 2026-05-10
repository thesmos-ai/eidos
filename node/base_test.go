// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
)

func TestBaseNode_Pos(t *testing.T) {
	t.Parallel()

	t.Run("returns the SourcePos", func(t *testing.T) {
		t.Parallel()
		pos := position.At("a.go", 12, 1)
		b := &node.BaseNode{SourcePos: pos}
		if b.Pos() != pos {
			t.Fatalf("Pos = %+v, want %+v", b.Pos(), pos)
		}
	})
}

func TestBaseNode_Docs(t *testing.T) {
	t.Parallel()

	t.Run("returns the DocLines slice", func(t *testing.T) {
		t.Parallel()
		b := &node.BaseNode{DocLines: []string{"first", "second"}}
		got := b.Docs()
		if len(got) != 2 || got[0] != "first" || got[1] != "second" {
			t.Fatalf("Docs = %v", got)
		}
	})
}

func TestBaseNode_Directives(t *testing.T) {
	t.Parallel()

	t.Run("returns the DirectiveList slice", func(t *testing.T) {
		t.Parallel()
		d := directiveAt("mock", position.Pos{})
		b := &node.BaseNode{DirectiveList: []*directive.Directive{d}}
		got := b.Directives()
		if len(got) != 1 || got[0] != d {
			t.Fatalf("Directives = %v", got)
		}
	})
}

func TestBaseNode_Directive(t *testing.T) {
	t.Parallel()

	t.Run("returns the first matching directive", func(t *testing.T) {
		t.Parallel()
		first := directiveAt("mock", position.At("a.go", 1, 1))
		second := directiveAt("mock", position.At("a.go", 2, 1))
		b := &node.BaseNode{DirectiveList: []*directive.Directive{first, second}}
		if got := b.Directive("mock"); got != first {
			t.Fatalf("Directive returned the wrong instance")
		}
	})

	t.Run("returns nil when no match exists", func(t *testing.T) {
		t.Parallel()
		var b node.BaseNode
		if got := b.Directive("mock"); got != nil {
			t.Fatalf("Directive on empty list = %v, want nil", got)
		}
	})
}

func TestBaseNode_HasDirective(t *testing.T) {
	t.Parallel()

	t.Run("returns true when at least one match exists", func(t *testing.T) {
		t.Parallel()
		b := &node.BaseNode{DirectiveList: []*directive.Directive{directiveAt("mock", position.Pos{})}}
		if !b.HasDirective("mock") {
			t.Fatalf("HasDirective should be true")
		}
	})

	t.Run("returns false when no match exists", func(t *testing.T) {
		t.Parallel()
		var b node.BaseNode
		if b.HasDirective("mock") {
			t.Fatalf("HasDirective on empty list should be false")
		}
	})
}

func TestBaseNode_Meta(t *testing.T) {
	t.Parallel()

	t.Run("lazy-initialises a non-nil bag on first call", func(t *testing.T) {
		t.Parallel()
		var b node.BaseNode
		if b.MetaBag != nil {
			t.Fatalf("zero-value BaseNode should have nil MetaBag")
		}
		bag := b.Meta()
		if bag == nil {
			t.Fatalf("Meta should return a non-nil bag")
		}
		if b.MetaBag != bag {
			t.Fatalf("Meta should cache the lazily-allocated bag on the receiver")
		}
	})

	t.Run("returns the same bag on subsequent calls", func(t *testing.T) {
		t.Parallel()
		var b node.BaseNode
		first := b.Meta()
		second := b.Meta()
		if first != second {
			t.Fatalf("Meta should return the same instance on every call")
		}
	})
}
