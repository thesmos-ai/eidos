// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/writer"
)

func TestSortSlotByPlan(t *testing.T) {
	t.Parallel()

	t.Run("nil slot returns nil", func(t *testing.T) {
		t.Parallel()
		if writer.SortSlotByPlan(nil, nil) != nil {
			t.Fatalf("nil slot should return nil")
		}
	})

	t.Run("empty slot returns nil", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "X"}
		s := host.FieldsSlot()
		if got := writer.SortSlotByPlan(s, nil); got != nil {
			t.Fatalf("empty slot should return nil; got %+v", got)
		}
	})

	t.Run("orders items by plan position; ties preserve insertion order", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "X"}
		slot := host.FieldsSlot()
		// Insert in reverse plan order: c (pos 2) then b (pos 1) then a (pos 0).
		assertNoError(t, slot.Append(&emit.Field{Name: "c"}, emit.Provenance{SetBy: "c"}))
		assertNoError(t, slot.Append(&emit.Field{Name: "b"}, emit.Provenance{SetBy: "b"}))
		assertNoError(t, slot.Append(&emit.Field{Name: "a"}, emit.Provenance{SetBy: "a"}))
		plan := map[string]int{"a": 0, "b": 1, "c": 2}
		got := writer.SortSlotByPlan(slot, plan)
		names := []string{got[0].(*emit.Field).Name, got[1].(*emit.Field).Name, got[2].(*emit.Field).Name}
		if names[0] != "a" || names[1] != "b" || names[2] != "c" {
			t.Fatalf("plan-position sort mismatch: %v", names)
		}
	})

	t.Run("items from the same plugin keep their insertion order", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "X"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "first"}, emit.Provenance{SetBy: "p"}))
		assertNoError(t, slot.Append(&emit.Field{Name: "second"}, emit.Provenance{SetBy: "p"}))
		plan := map[string]int{"p": 0}
		got := writer.SortSlotByPlan(slot, plan)
		if got[0].(*emit.Field).Name != "first" || got[1].(*emit.Field).Name != "second" {
			t.Fatalf("same-plugin items should preserve insertion order; got %v",
				[]string{got[0].(*emit.Field).Name, got[1].(*emit.Field).Name})
		}
	})

	t.Run("items from unmapped plugins sort after every mapped item", func(t *testing.T) {
		t.Parallel()
		host := &emit.Struct{Name: "X"}
		slot := host.FieldsSlot()
		assertNoError(t, slot.Append(&emit.Field{Name: "unknown"}, emit.Provenance{SetBy: "stranger"}))
		assertNoError(t, slot.Append(&emit.Field{Name: "known"}, emit.Provenance{SetBy: "a"}))
		plan := map[string]int{"a": 0}
		got := writer.SortSlotByPlan(slot, plan)
		if got[0].(*emit.Field).Name != "known" || got[1].(*emit.Field).Name != "unknown" {
			t.Fatalf("mapped items should come first; got %v",
				[]string{got[0].(*emit.Field).Name, got[1].(*emit.Field).Name})
		}
	})
}
