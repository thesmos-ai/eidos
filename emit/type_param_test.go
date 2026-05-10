// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestTypeParam_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindTypeParam", func(t *testing.T) {
		t.Parallel()
		var tp emit.TypeParam
		if tp.Kind() != emit.KindTypeParam {
			t.Fatalf("Kind = %s, want %s", tp.Kind(), emit.KindTypeParam)
		}
	})
}

func TestTypeParam_IsConstrained(t *testing.T) {
	t.Parallel()

	t.Run("reports true when Constraint is set", func(t *testing.T) {
		t.Parallel()
		tp := &emit.TypeParam{Name: "T", Constraint: builtinRef("any")}
		if !tp.IsConstrained() {
			t.Fatalf("constrained param should report IsConstrained true")
		}
	})

	t.Run("reports false when Constraint is nil", func(t *testing.T) {
		t.Parallel()
		tp := &emit.TypeParam{Name: "T"}
		if tp.IsConstrained() {
			t.Fatalf("unconstrained param should report IsConstrained false")
		}
	})
}
