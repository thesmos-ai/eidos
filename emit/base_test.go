// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

func TestBaseEmit_Pos(t *testing.T) {
	t.Parallel()

	t.Run("returns the SourcePos", func(t *testing.T) {
		t.Parallel()
		pos := position.At("a.go", 12, 1)
		b := &emit.BaseEmit{SourcePos: pos}
		if b.Pos() != pos {
			t.Fatalf("Pos = %+v, want %+v", b.Pos(), pos)
		}
	})
}

func TestBaseEmit_Docs(t *testing.T) {
	t.Parallel()

	t.Run("returns the DocLines slice", func(t *testing.T) {
		t.Parallel()
		b := &emit.BaseEmit{DocLines: []string{"first", "second"}}
		got := b.Docs()
		if len(got) != 2 || got[0] != "first" || got[1] != "second" {
			t.Fatalf("Docs = %v", got)
		}
	})
}

func TestBaseEmit_Directives(t *testing.T) {
	t.Parallel()

	t.Run("returns the DirectiveList slice", func(t *testing.T) {
		t.Parallel()
		d := directiveAt("mock", position.Pos{})
		b := &emit.BaseEmit{DirectiveList: []*directive.Directive{d}}
		got := b.Directives()
		if len(got) != 1 || got[0] != d {
			t.Fatalf("Directives = %v", got)
		}
	})
}

func TestBaseEmit_Directive(t *testing.T) {
	t.Parallel()

	t.Run("returns the first matching directive", func(t *testing.T) {
		t.Parallel()
		first := directiveAt("mock", position.At("a.go", 1, 1))
		second := directiveAt("mock", position.At("a.go", 2, 1))
		b := &emit.BaseEmit{DirectiveList: []*directive.Directive{first, second}}
		if got := b.Directive("mock"); got != first {
			t.Fatalf("Directive returned the wrong instance")
		}
	})

	t.Run("returns nil when no match exists", func(t *testing.T) {
		t.Parallel()
		var b emit.BaseEmit
		if got := b.Directive("mock"); got != nil {
			t.Fatalf("Directive on empty list = %v, want nil", got)
		}
	})
}

func TestBaseEmit_HasDirective(t *testing.T) {
	t.Parallel()

	t.Run("returns true when at least one match exists", func(t *testing.T) {
		t.Parallel()
		b := &emit.BaseEmit{DirectiveList: []*directive.Directive{directiveAt("mock", position.Pos{})}}
		if !b.HasDirective("mock") {
			t.Fatalf("HasDirective should be true")
		}
	})

	t.Run("returns false when no match exists", func(t *testing.T) {
		t.Parallel()
		var b emit.BaseEmit
		if b.HasDirective("mock") {
			t.Fatalf("HasDirective on empty list should be false")
		}
	})
}

func TestBaseEmit_Meta(t *testing.T) {
	t.Parallel()

	t.Run("lazy-initialises a non-nil bag on first call", func(t *testing.T) {
		t.Parallel()
		var b emit.BaseEmit
		if b.MetaBag != nil {
			t.Fatalf("zero-value BaseEmit should have nil MetaBag")
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
		var b emit.BaseEmit
		first := b.Meta()
		second := b.Meta()
		if first != second {
			t.Fatalf("Meta should return the same instance on every call")
		}
	})
}

func TestBaseEmit_Origin(t *testing.T) {
	t.Parallel()

	t.Run("returns the OriginNode field", func(t *testing.T) {
		t.Parallel()
		src := &node.Struct{Name: "Source"}
		b := &emit.BaseEmit{OriginNode: src}
		if b.Origin() != src {
			t.Fatalf("Origin should return the configured OriginNode")
		}
	})

	t.Run("returns nil for synthetic emit values", func(t *testing.T) {
		t.Parallel()
		var b emit.BaseEmit
		if b.Origin() != nil {
			t.Fatalf("zero-value BaseEmit should report nil Origin")
		}
	})
}
