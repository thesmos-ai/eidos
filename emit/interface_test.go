// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func makeInterface() *emit.Interface {
	return &emit.Interface{
		Name:    "Repo",
		Package: "users",
		Methods: []*emit.Method{
			{Name: "Get"},
			{Name: "Save"},
		},
	}
}

func TestInterface_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindInterface", func(t *testing.T) {
		t.Parallel()
		var i emit.Interface
		if i.Kind() != emit.KindInterface {
			t.Fatalf("Kind = %s, want %s", i.Kind(), emit.KindInterface)
		}
	})
}

func TestInterface_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeInterface().QName(), "users.Repo")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, (&emit.Interface{Name: "Foo"}).QName(), "Foo")
	})
}

func TestInterface_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("reports true when type params declared", func(t *testing.T) {
		t.Parallel()
		i := makeInterface()
		i.TypeParams = []*emit.TypeParam{{Name: "T"}}
		if !i.IsGeneric() {
			t.Fatalf("generic interface should report IsGeneric true")
		}
	})

	t.Run("reports false when no type params declared", func(t *testing.T) {
		t.Parallel()
		if makeInterface().IsGeneric() {
			t.Fatalf("non-generic interface should report IsGeneric false")
		}
	})
}

func TestInterface_MethodByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching method", func(t *testing.T) {
		t.Parallel()
		got := makeInterface().MethodByName("Save")
		if got == nil || got.Name != "Save" {
			t.Fatalf("MethodByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makeInterface().MethodByName("missing") != nil {
			t.Fatalf("MethodByName(unknown) should be nil")
		}
	})
}

func TestInterface_MethodsWith(t *testing.T) {
	t.Parallel()

	t.Run("filters methods by predicate", func(t *testing.T) {
		t.Parallel()
		got := makeInterface().MethodsWith(func(m *emit.Method) bool { return m.Name == "Get" })
		if len(got) != 1 || got[0].Name != "Get" {
			t.Fatalf("MethodsWith mismatch: %+v", got)
		}
	})
}

func TestInterface_Slots(t *testing.T) {
	t.Parallel()

	t.Run("MethodsSlot, EmbedsSlot, and Slot are distinct and idempotent", func(t *testing.T) {
		t.Parallel()
		i := makeInterface()
		m1, m2 := i.MethodsSlot(), i.MethodsSlot()
		e1, e2 := i.EmbedsSlot(), i.EmbedsSlot()
		c1, c2 := i.Slot("custom"), i.Slot("custom")
		if m1 != m2 || e1 != e2 || c1 != c2 {
			t.Fatalf("slot lookups should be idempotent")
		}
		if m1 == e1 || m1 == c1 || e1 == c1 {
			t.Fatalf("slots must be distinct instances")
		}
	})
}
