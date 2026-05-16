// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mocktest

import (
	"strconv"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/plugins/generator/mock"
)

// testingT is the `*testing.T` external ref used everywhere a test
// or subtest closure takes its testing parameter.
//
//nolint:gochecknoglobals // immutable ref shared across emit calls
var testingT = emit.Ptr(emit.External("testing", "T"))

// emitTests builds the Test<MockName>(t *testing.T) function for
// one mock struct. The function body holds one `t.Run(<method>,
// …)` subtest per dispatch method, each containing two branch
// subtests (nil + override). Back-links to s as origin so the
// routing layer places the test file alongside the mock.
func (*Plugin) emitTests(pkg *builder.PackageBuilder, s *emit.Struct) {
	// The PackageBuilder's Anchor default already stamps the source
	// interface as Origin on every decl built through pkg — no
	// per-decl Origin call needed here.
	pkg.Function("Test"+s.Name, func(fn *builder.FunctionBuilder) {
		fn.Docs("Test" + s.Name + " exercises the nil-check dispatch on every mock method.")
		fn.Param("t", testingT, nil)

		body := make([]*emit.Stmt, 0, 1+len(s.Methods))
		body = append(body, tParallel())
		for _, m := range s.Methods {
			body = append(body, methodSubtest(s, m))
		}
		fn.Body(body...)
	})
}

// methodSubtest builds the `t.Run("<MethodName>", func(t *testing.T) {
// … })` statement covering one mock method, with the per-branch
// subtests nested in the closure body.
func methodSubtest(s *emit.Struct, m *emit.Method) *emit.Stmt {
	body := []*emit.Stmt{
		tParallel(),
		nilDispatchSubtest(s, m),
		overrideDispatchSubtest(s, m),
	}
	return tRun(m.Name, body)
}

// nilDispatchSubtest builds the `t.Run("nil dispatch", …)` subtest
// — declares a zero-value mock and calls the method to drive the
// `if m.OnX != nil` false branch.
func nilDispatchSubtest(s *emit.Struct, m *emit.Method) *emit.Stmt {
	body := make([]*emit.Stmt, 0, 3+len(m.Params))
	body = append(body, tParallel(), declareZeroMock(s))
	body = append(body, declareParams(m)...)
	body = append(body, callMockMethod(m))
	return tRun("nil dispatch", body)
}

// overrideDispatchSubtest builds the `t.Run("override dispatch",
// …)` subtest — installs a closure on the method's override field,
// calls the method, and asserts the closure ran. Drives the `if
// m.OnX != nil` true branch.
func overrideDispatchSubtest(s *emit.Struct, m *emit.Method) *emit.Stmt {
	field, _ := mock.MetaField.Get(m.Meta())
	body := make([]*emit.Stmt, 0, 5+len(m.Params))
	body = append(body, tParallel(), declareCalledFlag(), assignMockWithOverride(s, m, field))
	body = append(body, declareParams(m)...)
	body = append(body, callMockMethod(m), assertCalled(field))
	return tRun("override dispatch", body)
}

// tParallel returns the bare `t.Parallel()` statement used at the
// top of every test / subtest closure.
func tParallel() *emit.Stmt {
	return emit.NewExprStmt(emit.NewMethodCall(emit.NewIdent("t"), "Parallel"))
}

// tRun returns `t.Run(<name>, func(t *testing.T) { <body> })` —
// the subtest expression-statement used at every level of the
// emitted test tree.
func tRun(name string, body []*emit.Stmt) *emit.Stmt {
	closure := emit.NewFuncLit(
		[]*emit.Param{{Name: "t", Type: testingT}},
		nil,
		body,
	)
	return emit.NewExprStmt(emit.NewMethodCall(
		emit.NewIdent("t"), "Run",
		emit.NewLiteralString(name),
		closure,
	))
}

// declareZeroMock returns `var m <MockName>` — the receiver for
// nil-dispatch invocations. Auto-addressable so Go promotes it for
// the mock's pointer-receiver method set.
func declareZeroMock(s *emit.Struct) *emit.Stmt {
	return emit.NewVarStmt("m", mockTypeRef(s), nil)
}

// mockTypeRef returns the ref the test uses to name the mock
// struct. [emit.Internal] is the right choice because the Go
// backend resolves cross-package qualification at render time
// from the target's resolved [emit.Target.ImportPath] — this
// remains correct under any routing configuration (default
// `_test`-pkg shift, `+gen:out` sibling-package, explicit
// `pkg=` override) without mocktest knowing which one applies.
func mockTypeRef(s *emit.Struct) emit.Ref {
	return emit.Internal(s)
}

// declareCalledFlag returns `called := false` — the assertion
// witness the override closure flips inside the override-dispatch
// subtest.
func declareCalledFlag() *emit.Stmt {
	return emit.NewAssign(
		[]*emit.Expr{emit.NewIdent("called")},
		":=",
		[]*emit.Expr{emit.NewLiteralBool(false)},
	)
}

// declareParams returns one `var <name> <type>` statement per
// method parameter. Zero values are sufficient for the dispatch
// branches — the mock body never inspects parameter content, so
// any type-correct value drives the same branch the test targets.
func declareParams(m *emit.Method) []*emit.Stmt {
	out := make([]*emit.Stmt, 0, len(m.Params))
	for i, p := range m.Params {
		out = append(out, emit.NewVarStmt(paramName(p.Name, i), p.Type, nil))
	}
	return out
}

// callMockMethod returns `m.<MethodName>(<args>)` — the bare
// expression statement that drives the dispatch. Discards return
// values; the branch coverage is what matters, not the result.
func callMockMethod(m *emit.Method) *emit.Stmt {
	args := make([]*emit.Expr, 0, len(m.Params))
	for i, p := range m.Params {
		args = append(args, emit.NewIdent(paramName(p.Name, i)))
	}
	return emit.NewExprStmt(emit.NewMethodCall(emit.NewIdent("m"), m.Name, args...))
}

// assignMockWithOverride returns `m := <MockName>{<field>:
// func(…) (…) { called = true; var _v0 …; return _v0, … }}` — the
// keyed composite literal that installs the override closure.
// Returns its zero values via per-return var declarations because
// emit's func literal does not carry named returns.
func assignMockWithOverride(s *emit.Struct, m *emit.Method, field string) *emit.Stmt {
	return emit.NewAssign(
		[]*emit.Expr{emit.NewIdent("m")},
		":=",
		[]*emit.Expr{emit.NewCompositeKeyed(
			mockTypeRef(s),
			[]string{field},
			[]*emit.Expr{overrideClosure(m)},
		)},
	)
}

// overrideClosure builds the `func(<params>) (<returns>) { called
// = true; return <zero values> }` literal assigned to the
// override field. Anonymous parameters keep the closure source
// concise — the test never reads them.
func overrideClosure(m *emit.Method) *emit.Expr {
	params := make([]*emit.Param, 0, len(m.Params))
	for _, p := range m.Params {
		params = append(params, &emit.Param{Name: "_", Type: p.Type})
	}
	returns := make([]emit.Ref, 0, len(m.Returns))
	for _, r := range m.Returns {
		returns = append(returns, r.Type)
	}

	body := make([]*emit.Stmt, 0, 2+len(m.Returns))
	body = append(body, emit.NewAssign(
		[]*emit.Expr{emit.NewIdent("called")},
		"=",
		[]*emit.Expr{emit.NewLiteralBool(true)},
	))
	retIdents := make([]*emit.Expr, 0, len(m.Returns))
	for i, r := range m.Returns {
		name := returnName(i)
		body = append(body, emit.NewVarStmt(name, r.Type, nil))
		retIdents = append(retIdents, emit.NewIdent(name))
	}
	body = append(body, emit.NewReturn(retIdents...))

	return emit.NewFuncLit(params, returns, body)
}

// assertCalled returns `if !called { t.Fatal("<Field> override was
// not invoked") }` — proves the override branch executed rather
// than just compiled.
func assertCalled(field string) *emit.Stmt {
	cond := emit.NewUnary("!", emit.NewIdent("called"))
	fail := emit.NewExprStmt(emit.NewMethodCall(
		emit.NewIdent("t"), "Fatal",
		emit.NewLiteralString(field+" override was not invoked"),
	))
	return emit.NewIf(cond, []*emit.Stmt{fail})
}

// paramName returns the in-test identifier for parameter index i —
// the source name when non-empty, or `arg<N>` for anonymous params.
// Mirrors the mock plugin's convention so generated tests align
// with the dispatch method's parameter list.
func paramName(name string, i int) string {
	if name != "" {
		return name
	}
	return "arg" + strconv.Itoa(i)
}

// returnName returns the per-return identifier for return index i.
// Aligns with the mock plugin's `_v<N>` convention.
func returnName(i int) string { return "_v" + strconv.Itoa(i) }
