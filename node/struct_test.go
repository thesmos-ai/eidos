// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func makeStruct() *node.Struct {
	return &node.Struct{
		Name:    "User",
		Package: "github.com/example/users",
		Fields: []*node.Field{
			{Name: "ID", Type: namedRef("", "string")},
			{Name: "Email", Type: namedRef("", "string")},
		},
		Methods: []*node.Method{
			{Name: "Validate"},
			{Name: "String"},
		},
	}
}

func TestStruct_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeStruct().QName(), "github.com/example/users.User")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		s := &node.Struct{Name: "Foo"}
		assertEqualString(t, s.QName(), "Foo")
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
		if got := makeStruct().FieldByName("missing"); got != nil {
			t.Fatalf("FieldByName(unknown) = %+v, want nil", got)
		}
	})
}

func TestStruct_FieldsWith(t *testing.T) {
	t.Parallel()

	t.Run("filters fields by predicate", func(t *testing.T) {
		t.Parallel()
		got := makeStruct().FieldsWith(func(f *node.Field) bool { return f.Name == "ID" })
		if len(got) != 1 || got[0].Name != "ID" {
			t.Fatalf("FieldsWith filter mismatch: %+v", got)
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
		if got := makeStruct().MethodByName("missing"); got != nil {
			t.Fatalf("MethodByName(unknown) = %+v, want nil", got)
		}
	})
}

func TestStruct_MethodsWith(t *testing.T) {
	t.Parallel()

	t.Run("filters methods by predicate", func(t *testing.T) {
		t.Parallel()
		got := makeStruct().MethodsWith(func(m *node.Method) bool { return m.Name == "String" })
		if len(got) != 1 || got[0].Name != "String" {
			t.Fatalf("MethodsWith filter mismatch: %+v", got)
		}
	})
}

func TestStruct_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("returns true when type params declared", func(t *testing.T) {
		t.Parallel()
		s := makeStruct()
		s.TypeParams = []*node.TypeParam{{Name: "T"}}
		if !s.IsGeneric() {
			t.Fatalf("generic struct should report IsGeneric true")
		}
	})

	t.Run("returns false when no type params declared", func(t *testing.T) {
		t.Parallel()
		if makeStruct().IsGeneric() {
			t.Fatalf("non-generic struct should report IsGeneric false")
		}
	})
}
