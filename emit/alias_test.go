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
