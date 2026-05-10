// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestVariable_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		v := &node.Variable{Name: "Default", Package: "github.com/example/cfg"}
		assertEqualString(t, v.QName(), "github.com/example/cfg.Default")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		v := &node.Variable{Name: "Default"}
		assertEqualString(t, v.QName(), "Default")
	})
}

func TestVariable_HasInitExpr(t *testing.T) {
	t.Parallel()

	t.Run("returns true when InitExpr is set", func(t *testing.T) {
		t.Parallel()
		v := &node.Variable{InitExpr: "42"}
		if !v.HasInitExpr() {
			t.Fatalf("HasInitExpr should be true")
		}
	})

	t.Run("returns false when InitExpr is empty", func(t *testing.T) {
		t.Parallel()
		var v node.Variable
		if v.HasInitExpr() {
			t.Fatalf("HasInitExpr should be false")
		}
	})
}

func TestVariable_HasDeclaredType(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Type is set", func(t *testing.T) {
		t.Parallel()
		v := &node.Variable{Type: namedRef("", "int")}
		if !v.HasDeclaredType() {
			t.Fatalf("HasDeclaredType should be true")
		}
	})

	t.Run("returns false when Type is nil", func(t *testing.T) {
		t.Parallel()
		var v node.Variable
		if v.HasDeclaredType() {
			t.Fatalf("HasDeclaredType should be false")
		}
	})
}
