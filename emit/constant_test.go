// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestConstant_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindConstant", func(t *testing.T) {
		t.Parallel()
		var c emit.Constant
		if c.Kind() != emit.KindConstant {
			t.Fatalf("Kind = %s, want %s", c.Kind(), emit.KindConstant)
		}
	})
}

func TestConstant_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		c := &emit.Constant{Name: "Pi", Package: "math"}
		assertEqualString(t, c.QName(), "math.Pi")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, (&emit.Constant{Name: "Pi"}).QName(), "Pi")
	})
}

func TestConstant_HasDeclaredType(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Type is set", func(t *testing.T) {
		t.Parallel()
		c := &emit.Constant{Type: builtinRef("float64")}
		if !c.HasDeclaredType() {
			t.Fatalf("constant with Type should report HasDeclaredType true")
		}
	})

	t.Run("returns false when Type is nil", func(t *testing.T) {
		t.Parallel()
		if (&emit.Constant{}).HasDeclaredType() {
			t.Fatalf("constant without Type should report HasDeclaredType false")
		}
	})
}
