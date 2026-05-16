// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func makeEnum() *node.Enum {
	return &node.Enum{
		Name:       "Status",
		Package:    "github.com/example/status",
		Underlying: namedRef("", "int"),
		Variants: []*node.EnumVariant{
			{Name: "StatusActive", Value: "1"},
			{Name: "StatusPaused", Value: "2"},
		},
	}
}

func TestEnum_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeEnum().QName(), "github.com/example/status.Status")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		e := &node.Enum{Name: "Status"}
		assertEqualString(t, e.QName(), "Status")
	})
}

func TestEnum_OwnerContract(t *testing.T) {
	t.Parallel()

	t.Run("OwnerName returns the bare identifier", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeEnum().OwnerName(), "Status")
	})

	t.Run("OwnerQName mirrors QName", func(t *testing.T) {
		t.Parallel()
		e := makeEnum()
		assertEqualString(t, e.OwnerQName(), e.QName())
	})
}

func TestEnum_VariantByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching variant", func(t *testing.T) {
		t.Parallel()
		got := makeEnum().VariantByName("StatusActive")
		if got == nil || got.Name != "StatusActive" {
			t.Fatalf("VariantByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makeEnum().VariantByName("missing"); got != nil {
			t.Fatalf("VariantByName(unknown) = %+v, want nil", got)
		}
	})
}

func TestEnum_VariantsWith(t *testing.T) {
	t.Parallel()

	t.Run("filters variants by predicate", func(t *testing.T) {
		t.Parallel()
		got := makeEnum().VariantsWith(func(v *node.EnumVariant) bool { return v.Value == "1" })
		if len(got) != 1 || got[0].Name != "StatusActive" {
			t.Fatalf("VariantsWith filter mismatch: %+v", got)
		}
	})
}

func TestEnum_HasUnderlying(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Underlying is set", func(t *testing.T) {
		t.Parallel()
		if !makeEnum().HasUnderlying() {
			t.Fatalf("HasUnderlying should be true when Underlying is set")
		}
	})

	t.Run("returns false when Underlying is nil", func(t *testing.T) {
		t.Parallel()
		var e node.Enum
		if e.HasUnderlying() {
			t.Fatalf("HasUnderlying should be false when Underlying is nil")
		}
	})
}
