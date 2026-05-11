// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/storefixture"
)

func TestBuilder_Enum(t *testing.T) {
	t.Parallel()

	t.Run("creates an enum with the configured name and package", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New().Package("users", "example.com/users").
			Enum("Status", nil)
		e := b.PackageNode().EnumByName("Status")
		if e == nil {
			t.Fatalf("Enum should be reachable by name")
		}
		requireQName(t, e.QName(), "example.com/users.Status")
	})

	t.Run("invokes the configuration callback exactly once", func(t *testing.T) {
		t.Parallel()
		var calls int
		storefixture.New().Enum("E", func(*storefixture.EnumBuilder) { calls++ })
		if calls != 1 {
			t.Fatalf("callback invocation count wrong: got %d, want 1", calls)
		}
	})
}

func TestEnumBuilder_Node(t *testing.T) {
	t.Parallel()

	t.Run("returns the enum backing the builder", func(t *testing.T) {
		t.Parallel()
		got := captureFirstEnum(t, func(*storefixture.EnumBuilder) {})
		if got == nil || got.Name != "E" {
			t.Fatalf("Node returned wrong enum: %+v", got)
		}
	})
}

func TestEnumBuilder_Pos(t *testing.T) {
	t.Parallel()

	t.Run("records the supplied position", func(t *testing.T) {
		t.Parallel()
		pos := position.At("e.go", 1, 1)
		got := captureFirstEnum(t, func(b *storefixture.EnumBuilder) { b.Pos(pos) })
		if !got.SourcePos.Equal(pos) {
			t.Fatalf("Pos not applied: %v", got.SourcePos)
		}
	})
}

func TestEnumBuilder_Docs(t *testing.T) {
	t.Parallel()

	t.Run("appends doc-comment lines in order", func(t *testing.T) {
		t.Parallel()
		got := captureFirstEnum(t, func(b *storefixture.EnumBuilder) {
			b.Docs("one").Docs("two")
		})
		if d := got.Docs(); len(d) != 2 || d[0] != "one" || d[1] != "two" {
			t.Fatalf("Docs order wrong: %+v", d)
		}
	})
}

func TestEnumBuilder_Directive(t *testing.T) {
	t.Parallel()

	t.Run("attaches the directive", func(t *testing.T) {
		t.Parallel()
		d := storefixture.Directive("expose")
		got := captureFirstEnum(t, func(b *storefixture.EnumBuilder) { b.Directive(d) })
		if !got.HasDirective("expose") {
			t.Fatalf("HasDirective should return true for expose")
		}
	})
}

func TestEnumBuilder_Underlying(t *testing.T) {
	t.Parallel()

	t.Run("records the underlying type", func(t *testing.T) {
		t.Parallel()
		typ := storefixture.Named("int")
		got := captureFirstEnum(t, func(b *storefixture.EnumBuilder) { b.Underlying(typ) })
		if !got.HasUnderlying() || got.Underlying != typ {
			t.Fatalf("Underlying not applied: %+v", got.Underlying)
		}
	})
}

func TestEnumBuilder_Variant(t *testing.T) {
	t.Parallel()

	t.Run("appends variants with owner wired in declaration order", func(t *testing.T) {
		t.Parallel()
		got := captureFirstEnum(t, func(b *storefixture.EnumBuilder) {
			b.Variant("Active", "1").Variant("Inactive", "2")
		})
		if len(got.Variants) != 2 {
			t.Fatalf("expected 2 variants; got %d", len(got.Variants))
		}
		first := got.VariantByName("Active")
		if first == nil || first.Value != "1" || first.Owner != got {
			t.Fatalf("first variant wiring wrong: %+v", first)
		}
		second := got.VariantByName("Inactive")
		if second == nil || second.Value != "2" || second.Owner != got {
			t.Fatalf("second variant wiring wrong: %+v", second)
		}
	})
}
