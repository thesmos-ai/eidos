// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestAlias_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindAlias", func(t *testing.T) {
		t.Parallel()
		var a emit.Alias
		if a.Kind() != emit.KindAlias {
			t.Fatalf("Kind = %s, want %s", a.Kind(), emit.KindAlias)
		}
	})
}

func TestAlias_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Name: "ID", Package: "users"}
		assertEqualString(t, a.QName(), "users.ID")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, (&emit.Alias{Name: "ID"}).QName(), "ID")
	})
}

func TestAlias_OwnerContract(t *testing.T) {
	t.Parallel()

	t.Run("OwnerName returns the bare identifier", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Name: "ID", Package: "users"}
		assertEqualString(t, a.OwnerName(), "ID")
	})

	t.Run("OwnerQName mirrors QName", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Name: "ID", Package: "users"}
		assertEqualString(t, a.OwnerQName(), a.QName())
	})
}

func TestAlias_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("reports true when type params declared", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Name: "Container", TypeParams: []*emit.TypeParam{{Name: "T"}}}
		if !a.IsGeneric() {
			t.Fatalf("generic alias should report IsGeneric true")
		}
	})

	t.Run("reports false otherwise", func(t *testing.T) {
		t.Parallel()
		if (&emit.Alias{Name: "X"}).IsGeneric() {
			t.Fatalf("non-generic alias should report IsGeneric false")
		}
	})
}

func TestAlias_MethodByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching method by name", func(t *testing.T) {
		t.Parallel()
		want := &emit.Method{Name: "Mul"}
		a := &emit.Alias{
			Methods: []*emit.Method{{Name: "Add"}, want, {Name: "Sub"}},
		}
		if got := a.MethodByName("Mul"); got != want {
			t.Fatalf("MethodByName(Mul) = %p, want %p", got, want)
		}
	})

	t.Run("returns nil when no method matches", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Methods: []*emit.Method{{Name: "Add"}}}
		if got := a.MethodByName("Missing"); got != nil {
			t.Fatalf("MethodByName(Missing) = %p, want nil", got)
		}
	})

	t.Run("returns nil for an alias with no methods", func(t *testing.T) {
		t.Parallel()
		var a emit.Alias
		if got := a.MethodByName("Any"); got != nil {
			t.Fatalf("MethodByName on empty Methods = %p, want nil", got)
		}
	})
}

func TestAlias_MethodsWith(t *testing.T) {
	t.Parallel()

	t.Run("returns methods matching the predicate in declaration order", func(t *testing.T) {
		t.Parallel()
		add := &emit.Method{Name: "Add"}
		mul := &emit.Method{Name: "Mul"}
		sub := &emit.Method{Name: "Sub"}
		a := &emit.Alias{Methods: []*emit.Method{add, mul, sub}}
		got := a.MethodsWith(func(m *emit.Method) bool {
			return m.Name == "Add" || m.Name == "Sub"
		})
		if len(got) != 2 || got[0] != add || got[1] != sub {
			t.Fatalf("MethodsWith returned %v, want [Add, Sub]", got)
		}
	})

	t.Run("returns an empty slice when nothing matches", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Methods: []*emit.Method{{Name: "Add"}}}
		got := a.MethodsWith(func(*emit.Method) bool { return false })
		if len(got) != 0 {
			t.Fatalf("MethodsWith returned %d entries, want 0", len(got))
		}
	})

	t.Run("returns an empty slice on an alias with no methods", func(t *testing.T) {
		t.Parallel()
		var a emit.Alias
		got := a.MethodsWith(func(*emit.Method) bool { return true })
		if len(got) != 0 {
			t.Fatalf("MethodsWith on empty Methods returned %d entries, want 0", len(got))
		}
	})
}

func TestAlias_MethodsSlot(t *testing.T) {
	t.Parallel()

	t.Run("returns the methods slot for cross-cutting injection", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Name: "Seconds"}
		slot := a.MethodsSlot()
		if slot == nil {
			t.Fatalf("MethodsSlot must not be nil")
		}
		if slot != a.MethodsSlot() {
			t.Fatalf("MethodsSlot must return the same instance on repeat calls")
		}
	})

	t.Run("methods slot rejects non-method contributions", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Name: "Seconds"}
		// Appending a non-method entity returns the kind-mismatch
		// sentinel; the typed slot constrains contributions to
		// emit.KindMethod.
		err := a.MethodsSlot().Append(&emit.Param{Name: "x"}, emit.Provenance{SetBy: "test"})
		if err == nil {
			t.Fatalf("expected kind-mismatch error for non-method contribution")
		}
	})
}

func TestAlias_Slot(t *testing.T) {
	t.Parallel()

	t.Run("returns a kind-unconstrained slot under the given name", func(t *testing.T) {
		t.Parallel()
		a := &emit.Alias{Name: "Seconds"}
		slot := a.Slot("custom")
		if slot == nil {
			t.Fatalf("Slot must not be nil")
		}
		if slot != a.Slot("custom") {
			t.Fatalf("Slot must return the same instance on repeat calls")
		}
		// Unconstrained slot accepts heterogeneous contributions.
		if err := slot.Append(&emit.Param{Name: "p"}, emit.Provenance{SetBy: "test"}); err != nil {
			t.Fatalf("unconstrained slot rejected a Param: %v", err)
		}
	})
}
