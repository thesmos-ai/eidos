// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestTypeParam_IsConstrained(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Constraint is set", func(t *testing.T) {
		t.Parallel()
		tp := node.TypeParam{Constraint: namedRef("", "any")}
		if !tp.IsConstrained() {
			t.Fatalf("Constraint set should report constrained")
		}
	})

	t.Run("returns false when Constraint is nil", func(t *testing.T) {
		t.Parallel()
		var tp node.TypeParam
		if tp.IsConstrained() {
			t.Fatalf("nil Constraint should not be constrained")
		}
	})
}
