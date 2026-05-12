// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

// TestReturn_Kind pins the [emit.KindReturn] discriminator the
// Node interface dispatches on. Backends type-switch over emit
// values via Kind(); the test guards against an accidental
// rename of the constant.
func TestReturn_Kind(t *testing.T) {
	t.Parallel()

	r := &emit.Return{Name: "n", Type: emit.Builtin("int")}
	if r.Kind() != emit.KindReturn {
		t.Fatalf("Kind = %q, want %q", r.Kind(), emit.KindReturn)
	}
}

// TestAnonReturns covers the convenience shape: each supplied type
// produces a fresh [*emit.Return] with the empty name and the
// passed-through Type. The variadic accepts zero arguments — the
// returned slice is non-nil but empty in that case.
func TestAnonReturns(t *testing.T) {
	t.Parallel()

	t.Run("wraps each type in an unnamed return", func(t *testing.T) {
		t.Parallel()
		got := emit.AnonReturns(emit.Builtin("int"), emit.Builtin("error"))
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		for i, r := range got {
			if r.Name != "" {
				t.Fatalf("got[%d].Name = %q, want empty", i, r.Name)
			}
		}
	})

	t.Run("zero arguments returns an empty slice", func(t *testing.T) {
		t.Parallel()
		got := emit.AnonReturns()
		if len(got) != 0 {
			t.Fatalf("zero-arg AnonReturns should return an empty slice; got %v", got)
		}
	})
}
