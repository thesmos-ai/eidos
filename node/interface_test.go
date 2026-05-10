// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func makeInterface() *node.Interface {
	return &node.Interface{
		Name:    "Repo",
		Package: "github.com/example/repo",
		Methods: []*node.Method{
			{Name: "Get"},
			{Name: "Save"},
		},
	}
}

func TestInterface_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeInterface().QName(), "github.com/example/repo.Repo")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		i := &node.Interface{Name: "Foo"}
		assertEqualString(t, i.QName(), "Foo")
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
		if got := makeInterface().MethodByName("missing"); got != nil {
			t.Fatalf("MethodByName(unknown) = %+v, want nil", got)
		}
	})
}

func TestInterface_MethodsWith(t *testing.T) {
	t.Parallel()

	t.Run("filters methods by predicate", func(t *testing.T) {
		t.Parallel()
		got := makeInterface().MethodsWith(func(m *node.Method) bool { return m.Name == "Get" })
		if len(got) != 1 || got[0].Name != "Get" {
			t.Fatalf("MethodsWith filter mismatch: %+v", got)
		}
	})
}

func TestInterface_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("returns true when type params declared", func(t *testing.T) {
		t.Parallel()
		i := makeInterface()
		i.TypeParams = []*node.TypeParam{{Name: "T"}}
		if !i.IsGeneric() {
			t.Fatalf("generic interface should report IsGeneric true")
		}
	})

	t.Run("returns false when no type params declared", func(t *testing.T) {
		t.Parallel()
		if makeInterface().IsGeneric() {
			t.Fatalf("non-generic interface should report IsGeneric false")
		}
	})
}
