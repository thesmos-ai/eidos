// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestVariable_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindVariable", func(t *testing.T) {
		t.Parallel()
		var v emit.Variable
		if v.Kind() != emit.KindVariable {
			t.Fatalf("Kind = %s, want %s", v.Kind(), emit.KindVariable)
		}
	})
}

func TestVariable_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		v := &emit.Variable{Name: "Default", Package: "users"}
		assertEqualString(t, v.QName(), "users.Default")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, (&emit.Variable{Name: "Default"}).QName(), "Default")
	})
}

func TestVariable_HasInit(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Init is non-nil", func(t *testing.T) {
		t.Parallel()
		v := &emit.Variable{Init: emit.NewLiteralInt(0)}
		if !v.HasInit() {
			t.Fatalf("variable with Init should report HasInit true")
		}
	})

	t.Run("returns false when Init is nil", func(t *testing.T) {
		t.Parallel()
		if (&emit.Variable{}).HasInit() {
			t.Fatalf("variable without Init should report HasInit false")
		}
	})
}

func TestVariable_HasDeclaredType(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Type is set", func(t *testing.T) {
		t.Parallel()
		v := &emit.Variable{Type: builtinRef("int")}
		if !v.HasDeclaredType() {
			t.Fatalf("variable with Type should report HasDeclaredType true")
		}
	})

	t.Run("returns false when Type is nil", func(t *testing.T) {
		t.Parallel()
		if (&emit.Variable{}).HasDeclaredType() {
			t.Fatalf("variable without Type should report HasDeclaredType false")
		}
	})
}
