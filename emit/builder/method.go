// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/contract"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// Method appends a top-level method to the package and runs fn
// against the resulting [MethodBuilder]. fn may be nil for an
// empty method declaration. The constructed method's Package
// field carries b.Node().Path so downstream routing can derive
// the rendered file's import path.
//
// Top-level methods are methods whose receiver type lives outside
// the emit graph (a source-side enum, a sentinel error declared
// by the user) — they cannot hang off an [emit.Struct] /
// [emit.Interface] / [emit.Alias] container's Methods slice and
// instead land on [emit.Package.Methods]. The PackageBuilder
// automatically stamps the method's Owner from the package
// builder's [Anchor]-supplied default origin when that origin
// satisfies [contract.Owner], and populates [emit.Method.OwnerRef]
// in lock-step so the resolved identity survives the cache-replay
// JSON round-trip.
func (b *PackageBuilder) Method(name string, fn func(*MethodBuilder)) *PackageBuilder {
	m := &emit.Method{
		BaseEmit: emit.BaseEmit{SetByName: b.ctx.SetBy()},
		Name:     name,
		Package:  b.pkg.Path,
		Target:   b.ctx.target,
	}
	applyBuilderDefaults(b, &m.BaseEmit)
	if owner, ok := b.defaultOrigin.(contract.Owner); ok {
		m.Owner = owner
		m.OwnerRef = contract.RefOf(owner)
	}
	mb := &MethodBuilder{ctx: b.ctx, m: m}
	if fn != nil {
		fn(mb)
	}
	b.pkg.Methods = append(b.pkg.Methods, m)
	return b
}

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

// Origin records the source [node.Node] this emit method was
// derived from — typically the source-side method or the source
// struct/interface whose surface the rendered method shadows.
// Pass nil to clear an existing origin.
func (b *MethodBuilder) Origin(n node.Node) *MethodBuilder {
	b.m.OriginNode = n
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

// TypeParam appends a generic type parameter to the method. fn
// (which may be nil) configures position / docs / directives on
// the resulting [emit.TypeParam].
func (b *MethodBuilder) TypeParam(
	name string,
	constraint *emit.Constraint,
	fn ...func(*TypeParamBuilder),
) *MethodBuilder {
	p := &emit.TypeParam{Name: name, Constraint: constraint, Owner: b.m}
	if len(fn) > 0 && fn[0] != nil {
		fn[0](&TypeParamBuilder{ctx: b.ctx, p: p})
	}
	b.m.TypeParams = append(b.m.TypeParams, p)
	return b
}

// Body sets the method's statement body. Existing body statements
// are replaced; cross-cutting contributions go through the prebody
// / postbody slots rather than this setter.
func (b *MethodBuilder) Body(stmts ...*emit.Stmt) *MethodBuilder {
	b.m.Body = stmts
	return b
}
