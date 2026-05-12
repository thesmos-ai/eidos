// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// InterfaceBuilder configures an [emit.Interface] as part of a
// [PackageBuilder]'s accumulating package. Spawned by
// [PackageBuilder.Interface] and handed to that method's callback;
// methods and embeds declared inside the callback receive their
// `Owner` back-pointer wired automatically.
type InterfaceBuilder struct {
	ctx *Context
	i   *emit.Interface
}

// Interface appends a new interface named name to the package and
// runs fn against its [InterfaceBuilder]. fn may be nil for an empty
// interface declaration. The constructed interface carries Package =
// b.Node().Path and Target = ctx.Target().
func (b *PackageBuilder) Interface(name string, fn func(*InterfaceBuilder)) *PackageBuilder {
	i := &emit.Interface{
		Name:    name,
		Package: b.pkg.Path,
		Target:  b.ctx.target,
	}
	ib := &InterfaceBuilder{ctx: b.ctx, i: i}
	if fn != nil {
		fn(ib)
	}
	b.pkg.Interfaces = append(b.pkg.Interfaces, i)
	return b
}

// Node returns the underlying [emit.Interface]. Use this accessor to
// set typed metadata on the interface, to pass the pointer to a
// downstream helper, or to capture the host reference for later
// cross-cutting contributions.
func (b *InterfaceBuilder) Node() *emit.Interface { return b.i }

// Target overrides the interface's [emit.Target].
func (b *InterfaceBuilder) Target(t emit.Target) *InterfaceBuilder {
	b.i.Target = t
	return b
}

// Pos overrides the interface's source position.
func (b *InterfaceBuilder) Pos(p position.Pos) *InterfaceBuilder {
	b.i.SourcePos = p
	return b
}

// Docs appends doc-comment lines preserved verbatim above the
// interface declaration.
func (b *InterfaceBuilder) Docs(lines ...string) *InterfaceBuilder {
	b.i.DocLines = append(b.i.DocLines, lines...)
	return b
}

// Directive attaches d to the interface's directive list.
func (b *InterfaceBuilder) Directive(d *directive.Directive) *InterfaceBuilder {
	b.i.DirectiveList = append(b.i.DirectiveList, d)
	return b
}

// Method appends a method to the interface (no body — interface
// methods are signatures only). The method's Owner is wired to the
// interface automatically.
func (b *InterfaceBuilder) Method(name string, fn func(*MethodBuilder)) *InterfaceBuilder {
	m := &emit.Method{Name: name, Owner: b.i}
	if fn != nil {
		fn(&MethodBuilder{ctx: b.ctx, m: m})
	}
	b.i.Methods = append(b.i.Methods, m)
	return b
}

// Embed appends an embedded type to the interface. Used for
// interface composition (`io.Reader`-style embedding inside another
// interface).
func (b *InterfaceBuilder) Embed(t emit.Ref, fn func(*EmbedBuilder)) *InterfaceBuilder {
	e := &emit.Embed{Type: t, Owner: b.i}
	if fn != nil {
		fn(&EmbedBuilder{ctx: b.ctx, e: e})
	}
	b.i.Embeds = append(b.i.Embeds, e)
	return b
}

// TypeParam appends a generic type parameter to the interface.
// fn (which may be nil) configures position / docs / directives on
// the resulting [emit.TypeParam].
func (b *InterfaceBuilder) TypeParam(
	name string,
	constraint *emit.Constraint,
	fn ...func(*TypeParamBuilder),
) *InterfaceBuilder {
	p := &emit.TypeParam{Name: name, Constraint: constraint, Owner: b.i}
	if len(fn) > 0 && fn[0] != nil {
		fn[0](&TypeParamBuilder{ctx: b.ctx, p: p})
	}
	b.i.TypeParams = append(b.i.TypeParams, p)
	return b
}
