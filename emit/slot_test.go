// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestSlot_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindSlot", func(t *testing.T) {
		t.Parallel()
		s := &emit.Slot{}
		if s.Kind() != emit.KindSlot {
			t.Fatalf("Kind = %s, want %s", s.Kind(), emit.KindSlot)
		}
	})
}

func TestSlot_Append(t *testing.T) {
	t.Parallel()

	t.Run("appends item with provenance and increments Len", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		err := slot.Append(&emit.Field{Name: "ID"}, emit.Provenance{SetBy: "id-gen"})
		assertNoError(t, err)
		if slot.Len() != 1 {
			t.Fatalf("Len = %d, want 1", slot.Len())
		}
	})

	t.Run("rejects an item whose Kind does not match ElemKind", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		err := slot.Append(&emit.Method{Name: "Save"}, emit.Provenance{})
		if !errors.Is(err, emit.ErrSlotElementType) {
			t.Fatalf("Append should return ErrSlotElementType; got %v", err)
		}
		if !strings.Contains(err.Error(), "fields") {
			t.Fatalf("error should mention slot name; got %q", err.Error())
		}
	})

	t.Run("rejects nil item when ElemKind is set", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		if err := slot.Append(nil, emit.Provenance{}); !errors.Is(err, emit.ErrSlotElementType) {
			t.Fatalf("nil item should be rejected when ElemKind is set; got %v", err)
		}
	})

	t.Run("accepts heterogeneous items when ElemKind is empty", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.Slot("custom")
		if err := slot.Append(&emit.Field{Name: "X"}, emit.Provenance{}); err != nil {
			t.Fatalf("Append should accept any item; got %v", err)
		}
		if err := slot.Append(&emit.Method{Name: "M"}, emit.Provenance{}); err != nil {
			t.Fatalf("Append should accept any item; got %v", err)
		}
	})
}

func TestSlot_Prepend(t *testing.T) {
	t.Parallel()

	t.Run("inserts item at the head, shifting existing items right", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		first := &emit.Field{Name: "First"}
		second := &emit.Field{Name: "Second"}
		assertNoError(t, slot.Append(first, emit.Provenance{SetBy: "a"}))
		assertNoError(t, slot.Prepend(second, emit.Provenance{SetBy: "b"}))
		if slot.At(0).(*emit.Field).Name != "Second" {
			t.Fatalf("Prepend should place item at index 0")
		}
	})

	t.Run("propagates kind-mismatch errors", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		err := slot.Prepend(&emit.Method{Name: "Save"}, emit.Provenance{})
		if !errors.Is(err, emit.ErrSlotElementType) {
			t.Fatalf("Prepend should propagate kind errors; got %v", err)
		}
	})
}

func TestSlot_InsertAt(t *testing.T) {
	t.Parallel()

	t.Run("inserts at the requested index", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "A"}, emit.Provenance{}))
		assertNoError(t, slot.Append(&emit.Field{Name: "B"}, emit.Provenance{}))
		assertNoError(t, slot.InsertAt(1, &emit.Field{Name: "Mid"}, emit.Provenance{}))
		if slot.Len() != 3 || slot.At(1).(*emit.Field).Name != "Mid" {
			t.Fatalf("InsertAt order incorrect: %+v", slot.Items)
		}
	})

	t.Run("inserts at the boundary index equal to Len", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "A"}, emit.Provenance{}))
		err := slot.InsertAt(1, &emit.Field{Name: "B"}, emit.Provenance{})
		assertNoError(t, err)
	})

	t.Run("rejects negative or out-of-range indexes", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		if err := slot.InsertAt(-1, &emit.Field{}, emit.Provenance{}); err == nil {
			t.Fatalf("expected error for negative index")
		}
		if err := slot.InsertAt(5, &emit.Field{}, emit.Provenance{}); err == nil {
			t.Fatalf("expected error for out-of-range index")
		}
	})

	t.Run("propagates kind-mismatch errors", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		err := slot.InsertAt(0, &emit.Method{Name: "Save"}, emit.Provenance{})
		if !errors.Is(err, emit.ErrSlotElementType) {
			t.Fatalf("InsertAt should propagate kind errors; got %v", err)
		}
	})
}

func TestSlot_At(t *testing.T) {
	t.Parallel()

	t.Run("returns the item at the supplied index", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		f := &emit.Field{Name: "X"}
		assertNoError(t, slot.Append(f, emit.Provenance{}))
		if slot.At(0) != f {
			t.Fatalf("At(0) should return the appended item")
		}
	})

	t.Run("returns nil for out-of-range indexes", func(t *testing.T) {
		t.Parallel()
		var s emit.Slot
		if s.At(0) != nil || s.At(-1) != nil {
			t.Fatalf("At should return nil for out-of-range indexes")
		}
	})
}

func TestSlot_ProvenanceAt(t *testing.T) {
	t.Parallel()

	t.Run("returns the recorded provenance", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "X"}, emit.Provenance{SetBy: "tagger"}))
		if slot.ProvenanceAt(0).SetBy != "tagger" {
			t.Fatalf("ProvenanceAt should return the recorded value")
		}
	})

	t.Run("returns the zero provenance for out-of-range indexes", func(t *testing.T) {
		t.Parallel()
		var s emit.Slot
		got := s.ProvenanceAt(0)
		if got != (emit.Provenance{}) {
			t.Fatalf("ProvenanceAt out-of-range = %+v, want zero value", got)
		}
	})
}

func TestSlot_LazyHostSlots(t *testing.T) {
	t.Parallel()

	t.Run("repeat Slot lookups return the same instance", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		first := host.Slot("custom")
		second := host.Slot("custom")
		if first != second {
			t.Fatalf("Slot lookup should be idempotent")
		}
	})
}

func TestSlot_InsertBefore(t *testing.T) {
	t.Parallel()

	t.Run("inserts immediately before the item matching the supplied ID", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "first"}, emit.Provenance{ID: "a"}))
		assertNoError(t, slot.Append(&emit.Field{Name: "last"}, emit.Provenance{ID: "c"}))
		assertNoError(t, slot.InsertBefore("c", &emit.Field{Name: "middle"}, emit.Provenance{ID: "b"}))
		names := []string{
			slot.At(0).(*emit.Field).Name,
			slot.At(1).(*emit.Field).Name,
			slot.At(2).(*emit.Field).Name,
		}
		if names[0] != "first" || names[1] != "middle" || names[2] != "last" {
			t.Fatalf("InsertBefore order mismatch: %v", names)
		}
	})

	t.Run("returns ErrProvenanceNotFound for unknown ID", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "x"}, emit.Provenance{ID: "a"}))
		err := slot.InsertBefore("nope", &emit.Field{Name: "y"}, emit.Provenance{})
		if !errors.Is(err, emit.ErrProvenanceNotFound) {
			t.Fatalf("InsertBefore on unknown ID should return ErrProvenanceNotFound; got %v", err)
		}
	})

	t.Run("returns ErrProvenanceNotFound for empty ID", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "x"}, emit.Provenance{}))
		err := slot.InsertBefore("", &emit.Field{Name: "y"}, emit.Provenance{})
		if !errors.Is(err, emit.ErrProvenanceNotFound) {
			t.Fatalf("InsertBefore(\"\") should return ErrProvenanceNotFound; got %v", err)
		}
	})
}

func TestSlot_InsertAfter(t *testing.T) {
	t.Parallel()

	t.Run("inserts immediately after the item matching the supplied ID", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "first"}, emit.Provenance{ID: "a"}))
		assertNoError(t, slot.Append(&emit.Field{Name: "last"}, emit.Provenance{ID: "c"}))
		assertNoError(t, slot.InsertAfter("a", &emit.Field{Name: "middle"}, emit.Provenance{ID: "b"}))
		names := []string{
			slot.At(0).(*emit.Field).Name,
			slot.At(1).(*emit.Field).Name,
			slot.At(2).(*emit.Field).Name,
		}
		if names[0] != "first" || names[1] != "middle" || names[2] != "last" {
			t.Fatalf("InsertAfter order mismatch: %v", names)
		}
	})

	t.Run("returns ErrProvenanceNotFound for unknown ID", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "x"}, emit.Provenance{ID: "a"}))
		err := slot.InsertAfter("nope", &emit.Field{Name: "y"}, emit.Provenance{})
		if !errors.Is(err, emit.ErrProvenanceNotFound) {
			t.Fatalf("InsertAfter on unknown ID should return ErrProvenanceNotFound; got %v", err)
		}
	})

	t.Run("returns ErrProvenanceNotFound for empty ID", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		err := slot.InsertAfter("", &emit.Field{Name: "y"}, emit.Provenance{})
		if !errors.Is(err, emit.ErrProvenanceNotFound) {
			t.Fatalf("InsertAfter(\"\") should return ErrProvenanceNotFound; got %v", err)
		}
	})

	t.Run("InsertAfter the last item appends to the end", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "User"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "only"}, emit.Provenance{ID: "a"}))
		assertNoError(t, slot.InsertAfter("a", &emit.Field{Name: "appended"}, emit.Provenance{}))
		if slot.Len() != 2 || slot.At(1).(*emit.Field).Name != "appended" {
			t.Fatalf("InsertAfter last should land at the end")
		}
	})
}
