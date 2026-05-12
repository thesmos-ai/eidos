// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestStmtConstructors covers each [builder] statement constructor
// by pinning the produced [emit.StmtKind]. The constructors
// forward to emit; this test guards against silent renaming or
// mis-wiring.
func TestStmtConstructors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  *emit.Stmt
		want emit.StmtKind
	}{
		{"Block", builder.Block(), emit.StmtBlock},
		{"ExprStmt", builder.ExprStmt(builder.Ident("x")), emit.StmtExpr},
		{
			"Assign",
			builder.Assign([]*emit.Expr{builder.Ident("x")}, "=", []*emit.Expr{builder.Int(1)}),
			emit.StmtAssign,
		},
		{"Return", builder.Return(), emit.StmtReturn},
		{"If", builder.If(builder.Bool(true), nil), emit.StmtIf},
		{"IfElse", builder.IfElse(builder.Bool(true), nil, nil), emit.StmtIf},
		{"IfInit", builder.IfInit(builder.RawStmt(""), builder.Bool(true), nil, nil), emit.StmtIf},
		{"ForStmt", builder.ForStmt(builder.Bool(true), nil), emit.StmtFor},
		{"ForFullStmt", builder.ForFullStmt(nil, builder.Bool(true), nil, nil), emit.StmtFor},
		{"ForRange", builder.ForRange("k", "v", builder.Ident("items"), nil), emit.StmtForRange},
		{"Switch", builder.Switch(builder.Ident("x"), nil), emit.StmtSwitch},
		{"SwitchInit", builder.SwitchInit(builder.RawStmt(""), builder.Ident("x"), nil), emit.StmtSwitch},
		{"Case", builder.Case([]*emit.Expr{builder.Int(1)}, nil), emit.StmtCase},
		{"Default", builder.Default(nil), emit.StmtDefault},
		{"Defer", builder.Defer(builder.Call(builder.Ident("f"))), emit.StmtDefer},
		{"Go", builder.Go(builder.Call(builder.Ident("f"))), emit.StmtGo},
		{"Break", builder.Break(""), emit.StmtBreak},
		{"Continue", builder.Continue(""), emit.StmtContinue},
		{"Label", builder.Label("loop", builder.ForStmt(nil, nil)), emit.StmtLabel},
		{"VarStmt", builder.VarStmt("x", emit.Builtin("int"), nil), emit.StmtVar},
		{"ConstStmt", builder.ConstStmt("x", emit.Builtin("int"), builder.Int(1)), emit.StmtConst},
		{"RawStmt", builder.RawStmt("return"), emit.StmtRaw},
	}
	for _, tc := range cases {
		t.Run(tc.name+" yields expected StmtKind", func(t *testing.T) {
			t.Parallel()
			if tc.got == nil {
				t.Fatalf("constructor returned nil")
			}
			if tc.got.StmtKind != tc.want {
				t.Fatalf("StmtKind = %v, want %v", tc.got.StmtKind, tc.want)
			}
		})
	}
}
