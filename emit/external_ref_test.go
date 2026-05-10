// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestExternal(t *testing.T) {
	t.Parallel()

	t.Run("constructs a ref with package, name, and type args", func(t *testing.T) {
		t.Parallel()
		r := emit.External("context", "Context")
		assertEqualString(t, r.Package, "context")
		assertEqualString(t, r.Name, "Context")
		if len(r.TypeArgs) != 0 {
			t.Fatalf("expected no type args; got %d", len(r.TypeArgs))
		}
	})

	t.Run("captures supplied type args for generic refs", func(t *testing.T) {
		t.Parallel()
		r := emit.External("sync", "Map", emit.Builtin("string"), emit.Builtin("int"))
		if len(r.TypeArgs) != 2 {
			t.Fatalf("expected 2 type args; got %d", len(r.TypeArgs))
		}
	})
}

func TestExternalRef_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindExternalRef", func(t *testing.T) {
		t.Parallel()
		r := emit.External("io", "Reader")
		if r.Kind() != emit.KindExternalRef {
			t.Fatalf("Kind = %s, want %s", r.Kind(), emit.KindExternalRef)
		}
	})
}

func TestExternalRef_SatisfiesRef(t *testing.T) {
	t.Parallel()

	t.Run("ExternalRef satisfies the Ref interface", func(t *testing.T) {
		t.Parallel()
		var _ emit.Ref = emit.External("io", "Reader")
	})
}
