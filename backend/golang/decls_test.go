// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/testpipe"
	"go.thesmos.sh/eidos/emit"
)

// TestInterface_Render covers the emit.interface template: methods
// inlined as signatures, embeds rendered as bare type references,
// the empty-interface case, and the unsupported-Ref propagation
// path.
func TestInterface_Render(t *testing.T) {
	t.Parallel()

	t.Run("interface with method signatures renders inline", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Interfaces: []*emit.Interface{{
				Name: "Closer", Package: "x", Target: target,
				Methods: []*emit.Method{
					{Name: "Close", Returns: []emit.Ref{emit.Builtin("error")}},
				},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type Closer interface {") {
			t.Fatalf("interface decl missing; got:\n%s", body)
		}
		if !strings.Contains(string(body), "Close() error") {
			t.Fatalf("method signature missing; got:\n%s", body)
		}
	})

	t.Run("empty interface renders as bare braces", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Interfaces: []*emit.Interface{{Name: "Any", Package: "x", Target: target}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		bareBraces := strings.Contains(string(body), "type Any interface {\n}")
		tightBraces := strings.Contains(string(body), "type Any interface{}")
		if !bareBraces && !tightBraces {
			t.Fatalf("empty interface should render with empty braces; got:\n%s", body)
		}
	})

	t.Run("interface with embeds renders embedded type references", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Interfaces: []*emit.Interface{{
				Name: "Reader", Package: "x", Target: target,
				Embeds: []*emit.Embed{
					{Type: emit.External("io", "Closer")},
				},
				Methods: []*emit.Method{
					{
						Name:    "Read",
						Params:  []*emit.Param{{Name: "n", Type: emit.Builtin("int")}},
						Returns: []emit.Ref{emit.Builtin("int"), emit.Builtin("error")},
					},
				},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "io.Closer\n") {
			t.Fatalf("embedded type ref missing; got:\n%s", body)
		}
		if !strings.Contains(string(body), "Read(n int) (int, error)") {
			t.Fatalf("method signature with multi-return mismatched; got:\n%s", body)
		}
	})

	t.Run("interface with mixed-named params produces ErrMixedNamedParams diagnostic", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Interfaces: []*emit.Interface{{
				Name: "I", Package: "x", Target: target,
				Methods: []*emit.Method{
					{Name: "M", Params: []*emit.Param{
						{Name: "a", Type: emit.Builtin("int")},
						{Type: emit.Builtin("string")}, // anonymous — mixed
					}},
				},
			}},
		})
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("render must not produce output on mixed-named params")
		}
		if !diagnosticsContain(d, diag.Error, "mixes named and unnamed") {
			t.Fatalf("expected ErrMixedNamedParams diagnostic; got %+v", d.Diagnostics())
		}
	})
}

// TestAlias_Render covers the emit.alias template for both forms:
// the alias form (`type X = Y`) and the definition form (`type X Y`).
func TestAlias_Render(t *testing.T) {
	t.Parallel()

	t.Run("type definition form renders without '='", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Aliases: []*emit.Alias{{
				Name: "UserID", Package: "x", File: target,
				Target:  emit.Builtin("int"),
				IsAlias: false,
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type UserID int\n") {
			t.Fatalf("type-definition form mismatched; got:\n%s", body)
		}
	})

	t.Run("type alias form renders with '='", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Aliases: []*emit.Alias{{
				Name: "Bytes", Package: "x", File: target,
				Target:  emit.Builtin("byte"),
				IsAlias: true,
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type Bytes = byte\n") {
			t.Fatalf("alias form mismatched; got:\n%s", body)
		}
	})
}

// TestVariable_Render covers the emit.variable template across the
// three init combinations: declared type only, init only (type
// inferred), and both.
func TestVariable_Render(t *testing.T) {
	t.Parallel()

	t.Run("typed var without initialiser renders as 'var X T'", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Variables: []*emit.Variable{{
				Name: "Counter", Package: "x", Target: target,
				Type: emit.Builtin("int"),
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "var Counter int\n") {
			t.Fatalf("typed-no-init form mismatched; got:\n%s", body)
		}
	})

	t.Run("typed var with literal initialiser", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Variables: []*emit.Variable{{
				Name: "Greeting", Package: "x", Target: target,
				Type: emit.Builtin("string"),
				Init: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitString, RawText: "hello"},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "var Greeting string = \"hello\"\n") {
			t.Fatalf("typed-with-init form mismatched; got:\n%s", body)
		}
	})

	t.Run("inferred-type var renders without type slot", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Variables: []*emit.Variable{{
				Name: "MaxRetries", Package: "x", Target: target,
				Init: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitInt, RawText: "3"},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "var MaxRetries = 3\n") {
			t.Fatalf("inferred-type form mismatched; got:\n%s", body)
		}
	})
}

// TestConstant_Render covers the emit.constant template:
// untyped and typed constants, plus the iota case via ExprIdent.
func TestConstant_Render(t *testing.T) {
	t.Parallel()

	t.Run("untyped constant renders as 'const X = V'", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Constants: []*emit.Constant{{
				Name: "Pi", Package: "x", Target: target,
				Value: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitFloat, RawText: "3.14"},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "const Pi = 3.14\n") {
			t.Fatalf("untyped const form mismatched; got:\n%s", body)
		}
	})

	t.Run("typed constant renders with declared type slot", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Constants: []*emit.Constant{{
				Name: "Limit", Package: "x", Target: target,
				Type:  emit.Builtin("int"),
				Value: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitInt, RawText: "100"},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "const Limit int = 100\n") {
			t.Fatalf("typed const form mismatched; got:\n%s", body)
		}
	})

	t.Run("constant valued at iota uses ExprIdent", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Constants: []*emit.Constant{{
				Name: "First", Package: "x", Target: target,
				Value: &emit.Expr{ExprKind: emit.ExprIdent, Name: "iota"},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "const First = iota\n") {
			t.Fatalf("iota constant form mismatched; got:\n%s", body)
		}
	})
}

// TestRenderExpr_LiteralKinds pins each [emit.LiteralKind] variant
// the funcmap currently supports against its rendered form. Uniform
// "literal in → rendered string out" mapping makes a table test the
// natural fit.
func TestRenderExpr_LiteralKinds(t *testing.T) {
	t.Parallel()

	lit := func(kind emit.LiteralKind, raw string) *emit.Expr {
		return &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: kind, RawText: raw}
	}
	cases := []struct {
		name string
		lit  *emit.Expr
		want string
	}{
		{"string is re-quoted", lit(emit.LitString, "hi"), "\"hi\""},
		{"int passes through raw", lit(emit.LitInt, "42"), "42"},
		{"uint passes through raw", lit(emit.LitUint, "42"), "42"},
		{"float passes through raw", lit(emit.LitFloat, "1.5"), "1.5"},
		{"bool passes through raw", lit(emit.LitBool, "false"), "false"},
		{"nil renders the keyword", lit(emit.LitNil, ""), "nil"},
		{"rune wraps in single quotes", lit(emit.LitRune, "a"), "'a'"},
		{"raw passes through verbatim", lit(emit.LitRaw, "0x1p3"), "0x1p3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := renderConstantValue(t, tc.lit)
			want := "const K = " + tc.want
			if !strings.Contains(body, want) {
				t.Fatalf("body should contain %q; got:\n%s", want, body)
			}
		})
	}
}

// renderConstantValue builds a single-constant fixture whose Value
// is the supplied expression, renders it, and returns the rendered
// file body. Used by TestRenderExpr_LiteralKinds to assert the
// per-literal-kind shape through the public render path.
func renderConstantValue(t *testing.T, value *emit.Expr) string {
	t.Helper()
	ctx, mem, d := newBackendContext(t)
	target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
	addEmitPackage(t, ctx, &emit.Package{
		Name: "x", Path: "x",
		Constants: []*emit.Constant{{
			Name: "K", Package: "x", Target: target,
			Value: value,
		}},
	})
	body := assertRenderSucceeds(t, ctx, mem, d, target)
	return string(body)
}

// TestRenderExpr_NilGuard pins the nil-input guard: a Variable
// constructed without an Init renders with no `= …` clause. The
// renderExpr helper returns the empty string when invoked with nil,
// which the variable template skips via its `{{ if .Init }}` guard;
// the resulting output omits the assignment entirely.
func TestRenderExpr_NilGuard(t *testing.T) {
	t.Parallel()
	t.Run("variable without Init renders with no assignment", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Variables: []*emit.Variable{{
				Name: "X", Package: "x", Target: target,
				Type: emit.Builtin("int"),
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if strings.Contains(string(body), "=") {
			t.Fatalf("uninitialised var should have no '='; got:\n%s", body)
		}
	})
}

// TestRenderParams_Variadic covers the variadic-parameter branch of
// renderParams — `...` prefix on the last parameter's type.
func TestRenderParams_Variadic(t *testing.T) {
	t.Parallel()
	t.Run("variadic param renders with '...' prefix", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Interfaces: []*emit.Interface{{
				Name: "Printer", Package: "x", Target: target,
				Methods: []*emit.Method{
					{Name: "Print", Params: []*emit.Param{
						{Name: "args", Type: emit.Builtin("string"), Variadic: true},
					}},
				},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "Print(args ...string)") {
			t.Fatalf("variadic param mismatched; got:\n%s", body)
		}
	})
}

// TestRenderExpr_UnsupportedKinds covers the not-yet-implemented
// ExprKinds: anything beyond ExprLiteral and ExprIdent surfaces as
// an Error diagnostic via ErrUnsupportedExpr.
func TestRenderExpr_UnsupportedKinds(t *testing.T) {
	t.Parallel()

	t.Run("ExprBinary returns ErrUnsupportedExpr", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Variables: []*emit.Variable{{
				Name: "X", Package: "x", Target: target,
				Init: &emit.Expr{
					ExprKind: emit.ExprBinary, Op: "+",
					Left:  &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitInt, RawText: "1"},
					Right: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitInt, RawText: "2"},
				},
			}},
		})
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("render must not produce output on unsupported expr")
		}
		if !diagnosticsContain(d, diag.Error, "unsupported Expr") {
			t.Fatalf("expected ErrUnsupportedExpr diagnostic; got %+v", d.Diagnostics())
		}
	})

	t.Run("ErrUnsupportedExpr is exported", func(t *testing.T) {
		t.Parallel()
		if golang.ErrUnsupportedExpr == nil {
			t.Fatalf("ErrUnsupportedExpr must be exported and non-nil")
		}
		if !errors.Is(golang.ErrUnsupportedExpr, golang.ErrUnsupportedExpr) {
			t.Fatalf("ErrUnsupportedExpr must satisfy errors.Is reflexivity")
		}
	})
}

// TestDecls_Golden pins canonical output for each E1 decl kind.
func TestDecls_Golden(t *testing.T) {
	t.Parallel()

	t.Run("interface_simple — methods + embeds", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "iox", Filename: "iox.go", Package: "iox"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "iox", Path: "iox",
			Interfaces: []*emit.Interface{{
				BaseEmit: emit.BaseEmit{DocLines: []string{"Reader is a minimal byte reader."}},
				Name:     "Reader", Package: "iox", Target: target,
				Embeds: []*emit.Embed{{Type: emit.External("io", "Closer")}},
				Methods: []*emit.Method{
					{
						Name:    "Read",
						Params:  []*emit.Param{{Name: "p", Type: emit.Builtin("byte")}},
						Returns: []emit.Ref{emit.Builtin("int"), emit.Builtin("error")},
					},
					{Name: "Reset"},
				},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "interface_simple.go.golden"))
	})

	t.Run("alias_definition — type X int", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "id.go", Package: "users"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "users", Path: "users",
			Aliases: []*emit.Alias{{
				BaseEmit: emit.BaseEmit{DocLines: []string{"UserID is the canonical primary key."}},
				Name:     "UserID", Package: "users", File: target,
				Target: emit.Builtin("int"),
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "alias_definition.go.golden"))
	})

	t.Run("alias_alias — type X = Y", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Aliases: []*emit.Alias{{
				Name: "Bytes", Package: "x", File: target,
				Target:  emit.Builtin("byte"),
				IsAlias: true,
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "alias_alias.go.golden"))
	})

	t.Run("variable_combinations — typed/inferred/no-init", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "cfg", Filename: "cfg.go", Package: "cfg"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "cfg", Path: "cfg",
			Variables: []*emit.Variable{
				{
					BaseEmit: emit.BaseEmit{DocLines: []string{"Counter tracks invocations."}},
					Name:     "Counter", Package: "cfg", Target: target,
					Type: emit.Builtin("int"),
				},
				{
					Name: "Greeting", Package: "cfg", Target: target,
					Type: emit.Builtin("string"),
					Init: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitString, RawText: "hello"},
				},
				{
					Name: "MaxRetries", Package: "cfg", Target: target,
					Init: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitInt, RawText: "3"},
				},
			},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "variable_combinations.go.golden"))
	})

	t.Run("constant_combinations — untyped/typed/iota", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "cfg", Filename: "cfg.go", Package: "cfg"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "cfg", Path: "cfg",
			Constants: []*emit.Constant{
				{
					BaseEmit: emit.BaseEmit{DocLines: []string{"Pi is a mathematical constant."}},
					Name:     "Pi", Package: "cfg", Target: target,
					Value: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitFloat, RawText: "3.14"},
				},
				{
					Name: "Limit", Package: "cfg", Target: target,
					Type:  emit.Builtin("int"),
					Value: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitInt, RawText: "100"},
				},
				{
					Name: "Enabled", Package: "cfg", Target: target,
					Value: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitBool, RawText: "true"},
				},
			},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "constant_combinations.go.golden"))
	})
}
