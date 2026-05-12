// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestChain covers the chain-builder — every chain method extends
// the cursor, and Build returns the accumulated expression. The
// case names describe the final shape produced by the chain.
func TestChain(t *testing.T) {
	t.Parallel()

	t.Run("Sel / Call / Index / TypeAssert build a deep chain", func(t *testing.T) {
		t.Parallel()
		got := builder.Chain(builder.Ident("db")).
			Sel("Conn").
			Call("Query", builder.Str("SELECT *")).
			Index(builder.Int(0)).
			TypeAssert(emit.Builtin("string")).
			Build()
		if got.ExprKind != emit.ExprTypeAssert {
			t.Fatalf("outermost = %v, want ExprTypeAssert", got.ExprKind)
		}
		idx := got.Receiver
		if idx == nil || idx.ExprKind != emit.ExprIndex {
			t.Fatalf("expected ExprIndex one layer in; got %+v", idx)
		}
		call := idx.Receiver
		if call == nil || call.ExprKind != emit.ExprMethodCall {
			t.Fatalf("expected ExprMethodCall two layers in; got %+v", call)
		}
		sel := call.Receiver
		if sel == nil || sel.ExprKind != emit.ExprField {
			t.Fatalf("expected ExprField three layers in; got %+v", sel)
		}
	})

	t.Run("Deref and Addr nest the cursor", func(t *testing.T) {
		t.Parallel()
		// `*&x` round-trip exercises both Chain prefix-mode methods.
		got := builder.Chain(builder.Ident("x")).Addr().Deref().Build()
		if got.ExprKind != emit.ExprDeref {
			t.Fatalf("outer should be Deref; got %v", got.ExprKind)
		}
		inner := got.Receiver
		if inner == nil || inner.ExprKind != emit.ExprAddr {
			t.Fatalf("inner should be Addr; got %+v", inner)
		}
	})
}
