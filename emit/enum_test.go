// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func makeEnum() *emit.Enum {
	return &emit.Enum{
		Name:       "Status",
		Package:    "users",
		Underlying: builtinRef("int"),
		Variants: []*emit.EnumVariant{
			{Name: "Active"},
			{Name: "Inactive"},
		},
	}
}

func TestEnum_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindEnum", func(t *testing.T) {
		t.Parallel()
		var e emit.Enum
		if e.Kind() != emit.KindEnum {
			t.Fatalf("Kind = %s, want %s", e.Kind(), emit.KindEnum)
		}
	})
}

func TestEnum_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeEnum().QName(), "users.Status")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, (&emit.Enum{Name: "X"}).QName(), "X")
	})
}

func TestEnum_HasUnderlying(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Underlying is set", func(t *testing.T) {
		t.Parallel()
		if !makeEnum().HasUnderlying() {
			t.Fatalf("enum with Underlying should report HasUnderlying true")
		}
	})

	t.Run("returns false when Underlying is nil", func(t *testing.T) {
		t.Parallel()
		if (&emit.Enum{Name: "X"}).HasUnderlying() {
			t.Fatalf("enum without Underlying should report HasUnderlying false")
		}
	})
}

func TestEnum_VariantByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching variant", func(t *testing.T) {
		t.Parallel()
		got := makeEnum().VariantByName("Active")
		if got == nil || got.Name != "Active" {
			t.Fatalf("VariantByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makeEnum().VariantByName("missing") != nil {
			t.Fatalf("VariantByName(unknown) should be nil")
		}
	})
}

func TestEnum_VariantsWith(t *testing.T) {
	t.Parallel()

	t.Run("filters variants by predicate", func(t *testing.T) {
		t.Parallel()
		got := makeEnum().VariantsWith(func(v *emit.EnumVariant) bool { return v.Name == "Inactive" })
		if len(got) != 1 || got[0].Name != "Inactive" {
			t.Fatalf("VariantsWith mismatch: %+v", got)
		}
	})
}

func TestEnum_Slots(t *testing.T) {
	t.Parallel()

	t.Run("VariantsSlot and Slot are distinct and idempotent", func(t *testing.T) {
		t.Parallel()
		e := makeEnum()
		v1, v2 := e.VariantsSlot(), e.VariantsSlot()
		c1, c2 := e.Slot("custom"), e.Slot("custom")
		if v1 != v2 || c1 != c2 {
			t.Fatalf("slot lookups should be idempotent")
		}
		if v1 == c1 {
			t.Fatalf("VariantsSlot and custom Slot must differ")
		}
	})
}
