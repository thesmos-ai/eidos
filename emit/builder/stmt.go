// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import "go.thesmos.sh/eidos/emit"

// This file re-exports the [emit] statement constructors under
// shorter names. Same convention as [expr.go]: no semantics are
// added beyond forwarding to emit.
//
// [ForStmt] / [ForFullStmt] take a `Stmt` suffix because the
// package-level [For] symbol is the Context constructor.

// Block wraps stmts as a single [emit.Stmt] block statement.
func Block(stmts ...*emit.Stmt) *emit.Stmt { return emit.NewBlock(stmts...) }

// ExprStmt wraps an expression as a statement: `expr`.
func ExprStmt(e *emit.Expr) *emit.Stmt { return emit.NewExprStmt(e) }

// Assign builds an assignment statement:
// `targets[0], targets[1], ... op values[0], values[1], ...`. op
// is the assignment operator (`=`, `:=`, `+=`, `-=`, etc.).
func Assign(targets []*emit.Expr, op string, values []*emit.Expr) *emit.Stmt {
	return emit.NewAssign(targets, op, values)
}

// Return builds a return statement: `return values...`. Pass no
// arguments for a bare `return`.
func Return(values ...*emit.Expr) *emit.Stmt { return emit.NewReturn(values...) }

// If builds an if statement without an else branch:
// `if cond { then }`.
func If(cond *emit.Expr, then []*emit.Stmt) *emit.Stmt { return emit.NewIf(cond, then) }

// IfElse builds an if/else statement: `if cond { then } else { els }`.
func IfElse(cond *emit.Expr, then, els []*emit.Stmt) *emit.Stmt {
	return emit.NewIfElse(cond, then, els)
}

// IfInit builds an if statement with an init clause:
// `if init; cond { then } else { els }`. Pass nil for els to omit
// the else branch.
func IfInit(init *emit.Stmt, cond *emit.Expr, then, els []*emit.Stmt) *emit.Stmt {
	return emit.NewIfInit(init, cond, then, els)
}

// ForStmt builds a for-condition loop: `for cond { body }`. Pass
// nil for cond to produce the bare-`for { body }` infinite-loop
// form. The `Stmt` suffix disambiguates from the package-level
// [For] Context constructor.
func ForStmt(cond *emit.Expr, body []*emit.Stmt) *emit.Stmt { return emit.NewFor(cond, body) }

// ForFullStmt builds a C-style for loop:
// `for init; cond; post { body }`. Any of init / cond / post may
// be nil.
func ForFullStmt(init *emit.Stmt, cond *emit.Expr, post *emit.Stmt, body []*emit.Stmt) *emit.Stmt {
	return emit.NewForFull(init, cond, post, body)
}

// ForRange builds a range loop: `for key, value := range over { body }`.
// Pass an empty key or value name to omit it (Go's `for _, v` /
// `for k := range x` / `for range x` forms).
func ForRange(key, value string, over *emit.Expr, body []*emit.Stmt) *emit.Stmt {
	return emit.NewForRange(key, value, over, body)
}

// Switch builds a switch statement: `switch cond { cases... }`.
// Pass nil for cond for the bare-`switch { case ... }` form.
func Switch(cond *emit.Expr, cases []*emit.Stmt) *emit.Stmt { return emit.NewSwitch(cond, cases) }

// SwitchInit builds a switch with an init clause:
// `switch init; cond { cases... }`.
func SwitchInit(init *emit.Stmt, cond *emit.Expr, cases []*emit.Stmt) *emit.Stmt {
	return emit.NewSwitchInit(init, cond, cases)
}

// Case builds one case clause inside a switch:
// `case values...: body`.
func Case(values []*emit.Expr, body []*emit.Stmt) *emit.Stmt { return emit.NewCase(values, body) }

// Default builds the `default: body` clause inside a switch.
func Default(body []*emit.Stmt) *emit.Stmt { return emit.NewDefault(body) }

// Defer builds a defer statement: `defer call`.
func Defer(call *emit.Expr) *emit.Stmt { return emit.NewDefer(call) }

// Go builds a go statement: `go call`.
func Go(call *emit.Expr) *emit.Stmt { return emit.NewGo(call) }

// Break builds a break statement. Pass an empty label for the
// bare `break`.
func Break(label string) *emit.Stmt { return emit.NewBreak(label) }

// Continue builds a continue statement. Pass an empty label for
// the bare `continue`.
func Continue(label string) *emit.Stmt { return emit.NewContinue(label) }

// Label builds a labelled statement: `name: inner`.
func Label(name string, inner *emit.Stmt) *emit.Stmt { return emit.NewLabel(name, inner) }

// VarStmt builds a local `var` statement:
// `var name typ = init`. typ may be nil to infer from init.
func VarStmt(name string, typ emit.Ref, init *emit.Expr) *emit.Stmt {
	return emit.NewVarStmt(name, typ, init)
}

// ConstStmt builds a local `const` statement:
// `const name typ = value`. typ may be nil to infer from value.
func ConstStmt(name string, typ emit.Ref, value *emit.Expr) *emit.Stmt {
	return emit.NewConstStmt(name, typ, value)
}

// RawStmt builds a statement containing verbatim text. Escape hatch
// for language constructs the model doesn't yet represent.
func RawStmt(text string) *emit.Stmt { return emit.NewRawStmt(text) }
