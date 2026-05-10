// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestInternal(t *testing.T) {
	t.Parallel()

	t.Run("constructs a ref pointing at the supplied target", func(t *testing.T) {
		t.Parallel()
		s := &emit.Struct{Name: "Repo"}
		r := emit.Internal(s)
		if r.Target != s {
			t.Fatalf("Target should be the supplied node; got %v", r.Target)
		}
		if len(r.TypeArgs) != 0 {
			t.Fatalf("expected no type args; got %d", len(r.TypeArgs))
		}
	})

	t.Run("captures supplied type args for generic instantiations", func(t *testing.T) {
		t.Parallel()
		s := &emit.Struct{Name: "Container"}
		r := emit.Internal(s, emit.Builtin("int"))
		if len(r.TypeArgs) != 1 {
			t.Fatalf("expected 1 type arg; got %d", len(r.TypeArgs))
		}
	})
}

func TestTypeRef_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindTypeRef", func(t *testing.T) {
		t.Parallel()
		r := emit.Internal(&emit.Struct{Name: "X"})
		if r.Kind() != emit.KindTypeRef {
			t.Fatalf("Kind = %s, want %s", r.Kind(), emit.KindTypeRef)
		}
	})
}

func TestTypeRef_SatisfiesRef(t *testing.T) {
	t.Parallel()

	t.Run("TypeRef satisfies the Ref interface", func(t *testing.T) {
		t.Parallel()
		var _ emit.Ref = emit.Internal(&emit.Struct{Name: "X"})
	})
}
