// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import "go.thesmos.sh/eidos/emit"

// ChainBuilder accumulates a left-to-right expression chain —
// method calls, field selections, and indexing — terminating with
// [ChainBuilder.Build] to surface the resulting [*emit.Expr].
//
// The motivating use case is plugin code constructing deep
// fluent-API expressions: query builders, RPC clients, options
// chains. The structured `emit.NewMethodCall(emit.NewMethodCall(...))`
// form nests right-to-left, which inverts the reading order. The
// chain builder keeps the source order natural:
//
//	builder.Chain(builder.Ident("db")).
//	    Call("Query", builder.Str("SELECT *")).
//	    Sel("Result").
//	    Call("Scan", builder.Addr(builder.Ident("row"))).
//	    Build()
//
// ChainBuilder mutates only its receiver; callers may capture an
// intermediate state and branch (chain.Sel("A") and chain.Sel("B")
// produce two independent terminals) by chaining off the same
// receiver multiple times.
type ChainBuilder struct {
	cur *emit.Expr
}

// Chain returns a fresh [ChainBuilder] seeded with seed as the
// chain's left-most expression. Pass any [*emit.Expr] — typically
// an [Ident] for a package-level seed or a receiver expression.
func Chain(seed *emit.Expr) *ChainBuilder { return &ChainBuilder{cur: seed} }

// Sel extends the chain with a field selection: `<current>.name`.
// Returns the same builder for chaining.
func (b *ChainBuilder) Sel(name string) *ChainBuilder {
	b.cur = emit.NewField(b.cur, name)
	return b
}

// Call extends the chain with a method call:
// `<current>.method(args...)`.
func (b *ChainBuilder) Call(method string, args ...*emit.Expr) *ChainBuilder {
	b.cur = emit.NewMethodCall(b.cur, method, args...)
	return b
}

// Index extends the chain with index access:
// `<current>[index]`.
func (b *ChainBuilder) Index(index *emit.Expr) *ChainBuilder {
	b.cur = emit.NewIndex(b.cur, index)
	return b
}

// TypeAssert extends the chain with a type-assertion:
// `<current>.(asType)`.
func (b *ChainBuilder) TypeAssert(asType emit.Ref) *ChainBuilder {
	b.cur = emit.NewTypeAssert(b.cur, asType)
	return b
}

// Deref extends the chain with a pointer dereference:
// `*<current>`. Unlike the other Chain methods this prefixes
// rather than suffixes, but the builder keeps a single
// accumulating cursor so the resulting expression nests
// correctly.
func (b *ChainBuilder) Deref() *ChainBuilder {
	b.cur = emit.NewDeref(b.cur)
	return b
}

// Addr extends the chain with an address-of operation:
// `&<current>`. As with [ChainBuilder.Deref], prefix semantics.
func (b *ChainBuilder) Addr() *ChainBuilder {
	b.cur = emit.NewAddr(b.cur)
	return b
}

// Build returns the accumulated [*emit.Expr]. Callers typically
// pass it into a statement constructor or assign it to a Variable
// / Constant initialiser.
func (b *ChainBuilder) Build() *emit.Expr { return b.cur }
