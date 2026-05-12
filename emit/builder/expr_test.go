// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestExprConstructors covers each [builder] expression
// constructor by pinning its produced [emit.ExprKind] (and
// [emit.LitKind] for literals). The constructors forward to emit;
// this test guards against silent renaming or mis-wiring.
func TestExprConstructors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  *emit.Expr
		want emit.ExprKind
	}{
		{"Ident", builder.Ident("x"), emit.ExprIdent},
		{"Sel", builder.Sel(builder.Ident("x"), "f"), emit.ExprField},
		{"Index", builder.Index(builder.Ident("a"), builder.Int(0)), emit.ExprIndex},
		{"Slice", builder.Slice(builder.Ident("a"), nil, nil, nil), emit.ExprSlice},
		{"Binary", builder.Binary(builder.Int(1), "+", builder.Int(2)), emit.ExprBinary},
		{"Unary", builder.Unary("-", builder.Int(1)), emit.ExprUnary},
		{"Call", builder.Call(builder.Ident("f")), emit.ExprCall},
		{"CallGeneric", builder.CallGeneric(builder.Ident("f"), []emit.Ref{emit.Builtin("int")}), emit.ExprCall},
		{"MethodCall", builder.MethodCall(builder.Ident("x"), "M"), emit.ExprMethodCall},
		{"TypeAssert", builder.TypeAssert(builder.Ident("x"), emit.Builtin("int")), emit.ExprTypeAssert},
		{"New", builder.New(emit.Builtin("T")), emit.ExprNew},
		{"Make", builder.Make(emit.Builtin("T")), emit.ExprMake},
		{"Composite", builder.Composite(emit.Builtin("T"), nil), emit.ExprComposite},
		{"CompositeKeyed", builder.CompositeKeyed(emit.Builtin("T"), nil, nil), emit.ExprCompositeKeyed},
		{"FuncLit", builder.FuncLit(nil, nil, nil), emit.ExprFuncLit},
		{"Paren", builder.Paren(builder.Ident("x")), emit.ExprParen},
		{"Deref", builder.Deref(builder.Ident("x")), emit.ExprDeref},
		{"Addr", builder.Addr(builder.Ident("x")), emit.ExprAddr},
		{"RawExpr", builder.RawExpr("(x+1)"), emit.ExprRaw},
		{"Str", builder.Str("hi"), emit.ExprLiteral},
		{"Int", builder.Int(42), emit.ExprLiteral},
		{"Uint", builder.Uint(42), emit.ExprLiteral},
		{"Float", builder.Float(3.14), emit.ExprLiteral},
		{"Bool", builder.Bool(true), emit.ExprLiteral},
		{"Nil", builder.Nil(), emit.ExprLiteral},
		{"Rune", builder.Rune("a"), emit.ExprLiteral},
		{"LitRaw", builder.LitRaw("1u32"), emit.ExprLiteral},
	}
	for _, tc := range cases {
		t.Run(tc.name+" yields expected ExprKind", func(t *testing.T) {
			t.Parallel()
			if tc.got == nil {
				t.Fatalf("constructor returned nil")
			}
			if tc.got.ExprKind != tc.want {
				t.Fatalf("ExprKind = %v, want %v", tc.got.ExprKind, tc.want)
			}
		})
	}
}

// TestExprLiterals_LitKinds covers the [emit.LitKind] discriminator
// each literal constructor produces. ExprKind alone is the same
// across literals (ExprLiteral); the LitKind is what discriminates
// them.
func TestExprLiterals_LitKinds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  *emit.Expr
		want emit.LiteralKind
	}{
		{"Str", builder.Str("x"), emit.LitString},
		{"Int", builder.Int(1), emit.LitInt},
		{"Uint", builder.Uint(1), emit.LitUint},
		{"Float", builder.Float(1.0), emit.LitFloat},
		{"Bool", builder.Bool(true), emit.LitBool},
		{"Nil", builder.Nil(), emit.LitNil},
		{"Rune", builder.Rune("r"), emit.LitRune},
		{"LitRaw", builder.LitRaw("1u32"), emit.LitRaw},
	}
	for _, tc := range cases {
		t.Run(tc.name+" yields expected LitKind", func(t *testing.T) {
			t.Parallel()
			if tc.got.LitKind != tc.want {
				t.Fatalf("LitKind = %v, want %v", tc.got.LitKind, tc.want)
			}
		})
	}
}
