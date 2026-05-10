// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestParam_IsAnonymous(t *testing.T) {
	t.Parallel()

	t.Run("returns true when Name is empty", func(t *testing.T) {
		t.Parallel()
		var p node.Param
		if !p.IsAnonymous() {
			t.Fatalf("zero Name should be anonymous")
		}
	})

	t.Run("returns false when Name is set", func(t *testing.T) {
		t.Parallel()
		p := node.Param{Name: "ctx"}
		if p.IsAnonymous() {
			t.Fatalf("named param should not be anonymous")
		}
	})
}
