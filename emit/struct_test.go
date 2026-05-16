// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func makeStruct() *emit.Struct {
	return &emit.Struct{
		Name:    "User",
		Package: "users",
		Fields: []*emit.Field{
			{Name: "ID", Type: builtinRef("string")},
			{Name: "Email", Type: builtinRef("string")},
		},
		Methods: []*emit.Method{
			{Name: "Validate"},
			{Name: "String"},
		},
	}
}

func TestStruct_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindStruct", func(t *testing.T) {
		t.Parallel()
		var s emit.Struct
		if s.Kind() != emit.KindStruct {
			t.Fatalf("Kind = %s, want %s", s.Kind(), emit.KindStruct)
		}
	})
}

func TestStruct_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeStruct().QName(), "users.User")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, (&emit.Struct{Name: "Foo"}).QName(), "Foo")
	})
}

func TestStruct_OwnerContract(t *testing.T) {
	t.Parallel()

	t.Run("OwnerName returns the bare identifier", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeStruct().OwnerName(), "User")
	})

	t.Run("OwnerQName mirrors QName", func(t *testing.T) {
		t.Parallel()
		s := makeStruct()
		assertEqualString(t, s.OwnerQName(), s.QName())
	})
}

func TestStruct_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("reports true when type params declared", func(t *testing.T) {
		t.Parallel()
		s := makeStruct()
		s.TypeParams = []*emit.TypeParam{{Name: "T"}}
		if !s.IsGeneric() {
			t.Fatalf("generic struct should report IsGeneric true")
		}
	})

	t.Run("reports false when no type params declared", func(t *testing.T) {
		t.Parallel()
		if makeStruct().IsGeneric() {
			t.Fatalf("non-generic struct should report IsGeneric false")
		}
	})
}

func TestStruct_FieldByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching field", func(t *testing.T) {
		t.Parallel()
		got := makeStruct().FieldByName("Email")
		if got == nil || got.Name != "Email" {
			t.Fatalf("FieldByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makeStruct().FieldByName("missing") != nil {
			t.Fatalf("FieldByName(unknown) should be nil")
		}
	})
}

func TestStruct_FieldsWith(t *testing.T) {
	t.Parallel()

	t.Run("filters fields by predicate", func(t *testing.T) {
		t.Parallel()
		got := makeStruct().FieldsWith(func(f *emit.Field) bool { return f.Name == "ID" })
		if len(got) != 1 || got[0].Name != "ID" {
			t.Fatalf("FieldsWith mismatch: %+v", got)
		}
	})
}

func TestStruct_MethodByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching method", func(t *testing.T) {
		t.Parallel()
		got := makeStruct().MethodByName("Validate")
		if got == nil || got.Name != "Validate" {
			t.Fatalf("MethodByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makeStruct().MethodByName("missing") != nil {
			t.Fatalf("MethodByName(unknown) should be nil")
		}
	})
}

func TestStruct_MethodsWith(t *testing.T) {
	t.Parallel()

	t.Run("filters methods by predicate", func(t *testing.T) {
		t.Parallel()
		got := makeStruct().MethodsWith(func(m *emit.Method) bool { return m.Name == "String" })
		if len(got) != 1 || got[0].Name != "String" {
			t.Fatalf("MethodsWith mismatch: %+v", got)
		}
	})
}

func TestStruct_Slots(t *testing.T) {
	t.Parallel()

	t.Run("FieldsSlot, MethodsSlot, EmbedsSlot are distinct and idempotent", func(t *testing.T) {
		t.Parallel()
		s := makeStruct()
		f1, f2 := s.FieldsSlot(), s.FieldsSlot()
		m1, m2 := s.MethodsSlot(), s.MethodsSlot()
		e1, e2 := s.EmbedsSlot(), s.EmbedsSlot()
		if f1 != f2 || m1 != m2 || e1 != e2 {
			t.Fatalf("standard slot lookups should be idempotent")
		}
		if f1 == m1 || f1 == e1 || m1 == e1 {
			t.Fatalf("standard slots must be distinct instances")
		}
	})

	t.Run("custom Slot lookup is idempotent", func(t *testing.T) {
		t.Parallel()
		s := makeStruct()
		if a, b := s.Slot("custom"), s.Slot("custom"); a != b {
			t.Fatalf("custom Slot lookup should be idempotent")
		}
	})
}
