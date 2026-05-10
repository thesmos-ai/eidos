// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestField_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindField", func(t *testing.T) {
		t.Parallel()
		var f emit.Field
		if f.Kind() != emit.KindField {
			t.Fatalf("Kind = %s, want %s", f.Kind(), emit.KindField)
		}
	})
}

func TestField_HasTag(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Tag is set", func(t *testing.T) {
		t.Parallel()
		f := &emit.Field{Name: "ID", Tag: `json:"id"`}
		if !f.HasTag() {
			t.Fatalf("field with Tag should report HasTag true")
		}
	})

	t.Run("returns false when Tag is empty", func(t *testing.T) {
		t.Parallel()
		f := &emit.Field{Name: "ID"}
		if f.HasTag() {
			t.Fatalf("field without Tag should report HasTag false")
		}
	})
}

func TestField_Tags(t *testing.T) {
	t.Parallel()

	t.Run("lazily allocates the tags slot and returns the same instance on repeat calls", func(t *testing.T) {
		t.Parallel()
		f := &emit.Field{Name: "ID"}
		first := f.Tags()
		second := f.Tags()
		if first == nil {
			t.Fatalf("Tags should return a non-nil slot")
		}
		if first != second {
			t.Fatalf("repeat Tags calls should return the same instance")
		}
	})
}

func TestField_SlotByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the named slot, lazily creating it", func(t *testing.T) {
		t.Parallel()
		f := &emit.Field{Name: "ID"}
		s := f.SlotByName("custom")
		if s == nil {
			t.Fatalf("SlotByName should return a non-nil slot")
		}
		if f.SlotByName("custom") != s {
			t.Fatalf("SlotByName should be idempotent")
		}
	})
}
