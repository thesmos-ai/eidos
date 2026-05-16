// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestAlias_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		a := &node.Alias{Name: "UserID", Package: "github.com/example/users"}
		assertEqualString(t, a.QName(), "github.com/example/users.UserID")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		a := &node.Alias{Name: "UserID"}
		assertEqualString(t, a.QName(), "UserID")
	})
}

func TestAlias_OwnerContract(t *testing.T) {
	t.Parallel()

	t.Run("OwnerName returns the bare identifier", func(t *testing.T) {
		t.Parallel()
		a := &node.Alias{Name: "UserID", Package: "github.com/example/users"}
		assertEqualString(t, a.OwnerName(), "UserID")
	})

	t.Run("OwnerQName mirrors QName", func(t *testing.T) {
		t.Parallel()
		a := &node.Alias{Name: "UserID", Package: "github.com/example/users"}
		assertEqualString(t, a.OwnerQName(), a.QName())
	})
}

func TestAlias_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("returns true when type params declared", func(t *testing.T) {
		t.Parallel()
		a := &node.Alias{TypeParams: []*node.TypeParam{{Name: "T"}}}
		if !a.IsGeneric() {
			t.Fatalf("alias with TypeParams should report IsGeneric true")
		}
	})

	t.Run("returns false when no type params declared", func(t *testing.T) {
		t.Parallel()
		var a node.Alias
		if a.IsGeneric() {
			t.Fatalf("alias without TypeParams should report IsGeneric false")
		}
	})
}

func TestAlias_MethodByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching method by name", func(t *testing.T) {
		t.Parallel()
		want := &node.Method{Name: "Mul"}
		a := &node.Alias{Methods: []*node.Method{{Name: "Add"}, want, {Name: "Sub"}}}
		if got := a.MethodByName("Mul"); got != want {
			t.Fatalf("MethodByName(Mul) = %p, want %p", got, want)
		}
	})

	t.Run("returns nil when no method matches", func(t *testing.T) {
		t.Parallel()
		a := &node.Alias{Methods: []*node.Method{{Name: "Add"}}}
		if got := a.MethodByName("Missing"); got != nil {
			t.Fatalf("MethodByName(Missing) = %p, want nil", got)
		}
	})

	t.Run("returns nil for an alias with no methods", func(t *testing.T) {
		t.Parallel()
		var a node.Alias
		if got := a.MethodByName("Any"); got != nil {
			t.Fatalf("MethodByName on empty Methods = %p, want nil", got)
		}
	})
}

func TestAlias_MethodsWith(t *testing.T) {
	t.Parallel()

	t.Run("returns methods matching the predicate in declaration order", func(t *testing.T) {
		t.Parallel()
		add := &node.Method{Name: "Add"}
		mul := &node.Method{Name: "Mul"}
		sub := &node.Method{Name: "Sub"}
		a := &node.Alias{Methods: []*node.Method{add, mul, sub}}
		got := a.MethodsWith(func(m *node.Method) bool {
			return m.Name == "Add" || m.Name == "Sub"
		})
		if len(got) != 2 || got[0] != add || got[1] != sub {
			t.Fatalf("MethodsWith returned %v, want [Add, Sub]", got)
		}
	})

	t.Run("returns an empty slice when nothing matches", func(t *testing.T) {
		t.Parallel()
		a := &node.Alias{Methods: []*node.Method{{Name: "Add"}}}
		got := a.MethodsWith(func(*node.Method) bool { return false })
		if len(got) != 0 {
			t.Fatalf("MethodsWith returned %d entries, want 0", len(got))
		}
	})

	t.Run("returns an empty slice on an alias with no methods", func(t *testing.T) {
		t.Parallel()
		var a node.Alias
		got := a.MethodsWith(func(*node.Method) bool { return true })
		if len(got) != 0 {
			t.Fatalf("MethodsWith on empty Methods returned %d entries, want 0", len(got))
		}
	})
}
