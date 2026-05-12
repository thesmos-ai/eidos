// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// MethodBuilder configures an [emit.Method] as part of a host's
// methods. Spawned by [StructBuilder.Method],
// [InterfaceBuilder.Method], or [AliasBuilder.Method]. The method's
// Owner is wired by the spawning builder so handlers never need to
// touch the back-pointer.
//
// Interface methods carry signatures only — Receiver, Body, and
// receiver-related slots stay zero. Struct / alias methods set
// Receiver via [MethodBuilder.Receiver].
type MethodBuilder struct {
	ctx *Context
	m   *emit.Method
}

// Node returns the underlying [emit.Method].
func (b *MethodBuilder) Node() *emit.Method { return b.m }

// Pos overrides the method's source position.
func (b *MethodBuilder) Pos(p position.Pos) *MethodBuilder {
	b.m.SourcePos = p
	return b
}

// Docs appends doc-comment lines above the method declaration.
func (b *MethodBuilder) Docs(lines ...string) *MethodBuilder {
	b.m.DocLines = append(b.m.DocLines, lines...)
	return b
}

// Directive attaches d to the method's directive list.
func (b *MethodBuilder) Directive(d *directive.Directive) *MethodBuilder {
	b.m.DirectiveList = append(b.m.DirectiveList, d)
	return b
}

// Receiver sets the method's receiver name and type. Used for
// methods declared on structs and named aliases; interface methods
// leave both zero.
//
// Pass an empty receiverName for the blank-receiver form
// (`func (*T) M()`); pass nil receiverType for an
// interface-method-style signature with no receiver clause.
func (b *MethodBuilder) Receiver(receiverName string, receiverType emit.Ref) *MethodBuilder {
	b.m.ReceiverName = receiverName
	b.m.Receiver = receiverType
	return b
}

// Param appends a positional parameter to the method.
func (b *MethodBuilder) Param(name string, t emit.Ref, fn ...func(*ParamBuilder)) *MethodBuilder {
	p := &emit.Param{Name: name, Type: t, Owner: b.m}
	if len(fn) > 0 && fn[0] != nil {
		fn[0](&ParamBuilder{ctx: b.ctx, p: p})
	}
	b.m.Params = append(b.m.Params, p)
	return b
}

// Return appends one [emit.Return] slot. Pass an empty name for the
// anonymous-return form; named-return methods declare every slot
// with a non-empty name.
func (b *MethodBuilder) Return(t emit.Ref, name ...string) *MethodBuilder {
	r := &emit.Return{Type: t}
	if len(name) > 0 {
		r.Name = name[0]
	}
	b.m.Returns = append(b.m.Returns, r)
	return b
}

// TypeParam appends a generic type parameter to the method.
func (b *MethodBuilder) TypeParam(name string, constraint *emit.Constraint) *MethodBuilder {
	b.m.TypeParams = append(b.m.TypeParams, &emit.TypeParam{
		Name:       name,
		Constraint: constraint,
		Owner:      b.m,
	})
	return b
}

// Body sets the method's statement body. Existing body statements
// are replaced; cross-cutting contributions go through the prebody
// / postbody slots rather than this setter.
func (b *MethodBuilder) Body(stmts ...*emit.Stmt) *MethodBuilder {
	b.m.Body = stmts
	return b
}
