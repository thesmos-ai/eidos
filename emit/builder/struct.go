// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// StructBuilder configures an [emit.Struct] as part of a
// [PackageBuilder]'s accumulating package. Spawned by
// [PackageBuilder.Struct] and handed to that method's callback; the
// underlying struct is appended to the package after the callback
// returns. Fields, methods, and embeds declared inside the callback
// receive their `Owner` back-pointer wired automatically.
type StructBuilder struct {
	ctx *Context
	s   *emit.Struct
}

// Struct appends a new struct named name to the package and runs fn
// against its [StructBuilder]. Returns the package builder for
// chaining further decls; the struct itself is reachable via the
// builder's Node accessor inside fn (`s.Node()`).
//
// fn may be nil for an empty struct declaration. The constructed
// struct carries Package = b.Node().Path and Target = ctx.Target() —
// the conventional defaults — both overridable through the struct
// builder if a caller needs them rewired.
func (b *PackageBuilder) Struct(name string, fn func(*StructBuilder)) *PackageBuilder {
	s := &emit.Struct{
		Name:    name,
		Package: b.pkg.Path,
		Target:  b.ctx.target,
	}
	sb := &StructBuilder{ctx: b.ctx, s: s}
	if fn != nil {
		fn(sb)
	}
	b.pkg.Structs = append(b.pkg.Structs, s)
	return b
}

// Node returns the underlying [emit.Struct]. Use this accessor to
// set typed metadata on the struct, to pass the pointer to a
// downstream helper, or to capture the host reference for later
// cross-cutting contributions.
func (b *StructBuilder) Node() *emit.Struct { return b.s }

// Target overrides the struct's [emit.Target]. By default the struct
// inherits its target from the spawning [Context]; this helper is
// used when one struct in a multi-struct generation needs to land in
// a different file from the rest.
func (b *StructBuilder) Target(t emit.Target) *StructBuilder {
	b.s.Target = t
	return b
}

// Pos overrides the struct's source position. Generated structs
// typically carry a zero [position.Pos]; callers stamping a position
// (for instance, the position of the source directive that triggered
// generation) use this helper.
func (b *StructBuilder) Pos(p position.Pos) *StructBuilder {
	b.s.SourcePos = p
	return b
}

// Docs appends doc-comment lines preserved verbatim above the
// struct declaration. Lines beginning with `//go:` or `//nolint:` are
// passed through to the renderer as compile-time directives.
func (b *StructBuilder) Docs(lines ...string) *StructBuilder {
	b.s.DocLines = append(b.s.DocLines, lines...)
	return b
}

// Directive attaches d to the struct's directive list.
func (b *StructBuilder) Directive(d *directive.Directive) *StructBuilder {
	b.s.DirectiveList = append(b.s.DirectiveList, d)
	return b
}

// Field appends a named field of the supplied type. fn (which may be
// nil) configures tag, doc comments, line comment, and per-field
// slots; the field's Owner is wired to the host struct automatically.
func (b *StructBuilder) Field(name string, t emit.Ref, fn func(*FieldBuilder)) *StructBuilder {
	f := &emit.Field{Name: name, Type: t, Owner: b.s}
	if fn != nil {
		fn(&FieldBuilder{ctx: b.ctx, f: f})
	}
	b.s.Fields = append(b.s.Fields, f)
	return b
}

// Embed appends an embedded type to the struct. fn (which may be
// nil) configures docs / directives; the embed's Owner is wired to
// the host struct automatically.
func (b *StructBuilder) Embed(t emit.Ref, fn func(*EmbedBuilder)) *StructBuilder {
	e := &emit.Embed{Type: t, Owner: b.s}
	if fn != nil {
		fn(&EmbedBuilder{ctx: b.ctx, e: e})
	}
	b.s.Embeds = append(b.s.Embeds, e)
	return b
}

// Method appends a method declared on this struct and runs fn
// against its [MethodBuilder]. The method's Owner is wired to the
// struct automatically; the method inherits the struct's Target so
// `b.Target(...)` flows through to its methods.
func (b *StructBuilder) Method(name string, fn func(*MethodBuilder)) *StructBuilder {
	m := &emit.Method{Name: name, Owner: b.s}
	if fn != nil {
		fn(&MethodBuilder{ctx: b.ctx, m: m})
	}
	b.s.Methods = append(b.s.Methods, m)
	return b
}

// TypeParam appends a generic type parameter to the struct.
// constraint may be nil to denote the implicit-any bound.
func (b *StructBuilder) TypeParam(name string, constraint *emit.Constraint) *StructBuilder {
	b.s.TypeParams = append(b.s.TypeParams, &emit.TypeParam{
		Name:       name,
		Constraint: constraint,
		Owner:      b.s,
	})
	return b
}
