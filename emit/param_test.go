// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestParam_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindParam", func(t *testing.T) {
		t.Parallel()
		var p emit.Param
		if p.Kind() != emit.KindParam {
			t.Fatalf("Kind = %s, want %s", p.Kind(), emit.KindParam)
		}
	})
}

func TestParam_IsAnonymous(t *testing.T) {
	t.Parallel()

	t.Run("reports true when Name is empty", func(t *testing.T) {
		t.Parallel()
		p := &emit.Param{Type: builtinRef("int")}
		if !p.IsAnonymous() {
			t.Fatalf("Name-less param should be anonymous")
		}
	})

	t.Run("reports false when Name is set", func(t *testing.T) {
		t.Parallel()
		p := &emit.Param{Name: "x", Type: builtinRef("int")}
		if p.IsAnonymous() {
			t.Fatalf("named param should not be anonymous")
		}
	})
}
