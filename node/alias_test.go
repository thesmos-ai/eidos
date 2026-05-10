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
