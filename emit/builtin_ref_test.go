// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestBuiltin(t *testing.T) {
	t.Parallel()

	t.Run("constructs a ref with the supplied name", func(t *testing.T) {
		t.Parallel()
		r := emit.Builtin("string")
		assertEqualString(t, r.Name, "string")
	})
}

func TestBuiltinRef_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindBuiltinRef", func(t *testing.T) {
		t.Parallel()
		r := emit.Builtin("int")
		if r.Kind() != emit.KindBuiltinRef {
			t.Fatalf("Kind = %s, want %s", r.Kind(), emit.KindBuiltinRef)
		}
	})
}

func TestBuiltinRef_SatisfiesRef(t *testing.T) {
	t.Parallel()

	t.Run("BuiltinRef satisfies the Ref interface", func(t *testing.T) {
		t.Parallel()
		var _ emit.Ref = emit.Builtin("any")
	})
}
