// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import "go.thesmos.sh/eidos/emit"

// This file re-exports the [emit] expression constructors under
// shorter names so plugin authors can compose expression trees
// entirely against the `builder` package — no second import.
//
// The names follow Go's spelling where possible: `Ident`, `Call`,
// `Make`, `Index`, `Sel` (field selector), and so on. Where a
// builder-package symbol already owns the natural name (e.g. the
// [For] Context constructor), the expression / statement variant
// takes a disambiguating suffix.
//
// Wrappers carry no behaviour beyond forwarding — the semantics
// are defined entirely by [emit]. This is by design: a builder
// layer that paraphrases emit's semantics would inevitably drift.

// Ident builds an identifier expression: `name`.
func Ident(name string) *emit.Expr { return emit.NewIdent(name) }

// Sel builds a field-selector expression: `receiver.name`.
func Sel(receiver *emit.Expr, name string) *emit.Expr { return emit.NewField(receiver, name) }

// Index builds an index expression: `receiver[index]`.
func Index(receiver, index *emit.Expr) *emit.Expr { return emit.NewIndex(receiver, index) }

// Slice builds a slice expression. Pass nil for any of low / high /
// capacity to omit that bound; supply capacity for the three-index
// form `receiver[low:high:capacity]`.
func Slice(receiver, low, high, capacity *emit.Expr) *emit.Expr {
	return emit.NewSlice(receiver, low, high, capacity)
}

// Binary builds a binary expression: `left op right`.
func Binary(left *emit.Expr, op string, right *emit.Expr) *emit.Expr {
	return emit.NewBinary(left, op, right)
}

// Unary builds a unary expression: `op operand`.
func Unary(op string, operand *emit.Expr) *emit.Expr { return emit.NewUnary(op, operand) }

// Call builds a function call: `callee(args...)`.
func Call(callee *emit.Expr, args ...*emit.Expr) *emit.Expr { return emit.NewCall(callee, args...) }

// CallGeneric builds a generic function call:
// `callee[typeArgs...](args...)`.
func CallGeneric(callee *emit.Expr, typeArgs []emit.Ref, args ...*emit.Expr) *emit.Expr {
	return emit.NewCallGeneric(callee, typeArgs, args...)
}

// MethodCall builds a method call: `receiver.method(args...)`.
func MethodCall(receiver *emit.Expr, method string, args ...*emit.Expr) *emit.Expr {
	return emit.NewMethodCall(receiver, method, args...)
}

// TypeAssert builds a type-assertion expression: `receiver.(asType)`.
func TypeAssert(receiver *emit.Expr, asType emit.Ref) *emit.Expr {
	return emit.NewTypeAssert(receiver, asType)
}

// New builds a Go `new(T)` expression.
func New(typ emit.Ref) *emit.Expr { return emit.NewNew(typ) }

// Make builds a Go `make(T, args...)` expression.
func Make(typ emit.Ref, args ...*emit.Expr) *emit.Expr { return emit.NewMake(typ, args...) }

// Composite builds a composite literal with positional elements:
// `T{e1, e2, e3}`.
func Composite(typ emit.Ref, elements []*emit.Expr) *emit.Expr {
	return emit.NewComposite(typ, elements)
}

// CompositeKeyed builds a composite literal with keyed elements:
// `T{K1: V1, K2: V2}`. keys and elements must have the same length;
// the rendered output preserves the supplied key/value pairing.
func CompositeKeyed(typ emit.Ref, keys []string, elements []*emit.Expr) *emit.Expr {
	return emit.NewCompositeKeyed(typ, keys, elements)
}

// FuncLit builds a function-literal expression:
// `func(params) returns { body }`. Pass an empty returns slice for
// no return values; pass an empty body slice for an empty function
// literal.
func FuncLit(params []*emit.Param, returns []emit.Ref, body []*emit.Stmt) *emit.Expr {
	return emit.NewFuncLit(params, returns, body)
}

// Paren builds a parenthesised expression: `(inner)`.
func Paren(inner *emit.Expr) *emit.Expr { return emit.NewParen(inner) }

// Deref builds a pointer-dereference expression: `*target`.
func Deref(target *emit.Expr) *emit.Expr { return emit.NewDeref(target) }

// Addr builds an address-of expression: `&target`.
func Addr(target *emit.Expr) *emit.Expr { return emit.NewAddr(target) }

// RawExpr builds an expression containing verbatim text. Used as
// an escape hatch when no structured constructor fits the desired
// output (typically: language constructs the model doesn't yet
// represent).
func RawExpr(text string) *emit.Expr { return emit.NewRawExpr(text) }

// Literal constructors.

// Str builds a string-literal expression: `"value"`. The supplied
// value is Go-quoted at render time.
func Str(value string) *emit.Expr { return emit.NewLiteralString(value) }

// Int builds an integer-literal expression rendering as the
// decimal representation of i.
func Int(i int64) *emit.Expr { return emit.NewLiteralInt(i) }

// Uint builds an unsigned-integer-literal expression rendering as
// the decimal representation of u.
func Uint(u uint64) *emit.Expr { return emit.NewLiteralUint(u) }

// Float builds a floating-point-literal expression rendering as
// the Go-canonical representation of f.
func Float(f float64) *emit.Expr { return emit.NewLiteralFloat(f) }

// Bool builds a boolean-literal expression: `true` or `false`.
func Bool(b bool) *emit.Expr { return emit.NewLiteralBool(b) }

// Nil builds a nil-literal expression: `nil`.
func Nil() *emit.Expr { return emit.NewLiteralNil() }

// Rune builds a rune-literal expression: `'c'`. r is the rune
// content without the surrounding single-quote delimiters.
func Rune(r string) *emit.Expr { return emit.NewLiteralRune(r) }

// LitRaw builds a literal-expression containing verbatim text.
// Used when the rendered form of a literal doesn't match any of
// the typed literal kinds (typically: language-specific suffixes
// like Rust's `1u32`).
func LitRaw(text string) *emit.Expr { return emit.NewLiteralRaw(text) }
