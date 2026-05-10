// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestConstant_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		c := &node.Constant{Name: "Pi", Package: "github.com/example/math"}
		assertEqualString(t, c.QName(), "github.com/example/math.Pi")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		c := &node.Constant{Name: "Pi"}
		assertEqualString(t, c.QName(), "Pi")
	})
}

func TestConstant_HasDeclaredType(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Type is set", func(t *testing.T) {
		t.Parallel()
		c := &node.Constant{Type: namedRef("", "int")}
		if !c.HasDeclaredType() {
			t.Fatalf("HasDeclaredType should be true")
		}
	})

	t.Run("returns false when Type is nil", func(t *testing.T) {
		t.Parallel()
		var c node.Constant
		if c.HasDeclaredType() {
			t.Fatalf("HasDeclaredType should be false")
		}
	})
}
