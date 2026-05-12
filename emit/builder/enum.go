// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// EnumBuilder configures an [emit.Enum] as part of a
// [PackageBuilder]'s accumulating package. Spawned by
// [PackageBuilder.Enum]; variants declared inside the callback
// receive their `Owner` back-pointer wired automatically.
type EnumBuilder struct {
	ctx *Context
	e   *emit.Enum
}

// Enum appends a new enum named name to the package and runs fn
// against its [EnumBuilder]. The enum carries Package = b.Node().Path
// and Target = ctx.Target().
//
// underlying may be nil for an iota-driven enum without an explicit
// underlying type (the renderer auto-promotes a typed-iota block);
// non-nil callers typically pass `emit.Builtin("int")` or similar.
func (b *PackageBuilder) Enum(name string, underlying emit.Ref, fn func(*EnumBuilder)) *PackageBuilder {
	e := &emit.Enum{
		BaseEmit:   emit.BaseEmit{SetByName: b.ctx.SetBy()},
		Name:       name,
		Package:    b.pkg.Path,
		Underlying: underlying,
		Target:     b.ctx.target,
	}
	eb := &EnumBuilder{ctx: b.ctx, e: e}
	if fn != nil {
		fn(eb)
	}
	b.pkg.Enums = append(b.pkg.Enums, e)
	return b
}

// Node returns the underlying [emit.Enum].
func (b *EnumBuilder) Node() *emit.Enum { return b.e }

// Target overrides the enum's [emit.Target].
func (b *EnumBuilder) Target(t emit.Target) *EnumBuilder {
	b.e.Target = t
	return b
}

// Pos overrides the enum's source position.
func (b *EnumBuilder) Pos(p position.Pos) *EnumBuilder {
	b.e.SourcePos = p
	return b
}

// Origin records the source [node.Node] this emit enum was
// derived from. Pass nil to clear an existing origin.
func (b *EnumBuilder) Origin(n node.Node) *EnumBuilder {
	b.e.OriginNode = n
	return b
}

// Docs appends doc-comment lines above the enum declaration.
func (b *EnumBuilder) Docs(lines ...string) *EnumBuilder {
	b.e.DocLines = append(b.e.DocLines, lines...)
	return b
}

// Directive attaches d to the enum's directive list.
func (b *EnumBuilder) Directive(d *directive.Directive) *EnumBuilder {
	b.e.DirectiveList = append(b.e.DirectiveList, d)
	return b
}

// Variant appends a variant named name with the supplied value
// expression. Pass nil for value to let the renderer's iota
// auto-increment supply the value (the canonical Go-enum form).
// fn (which may be nil) configures docs / directives.
func (b *EnumBuilder) Variant(name string, value *emit.Expr, fn func(*EnumVariantBuilder)) *EnumBuilder {
	v := &emit.EnumVariant{Name: name, Value: value, Owner: b.e}
	if fn != nil {
		fn(&EnumVariantBuilder{ctx: b.ctx, v: v})
	}
	b.e.Variants = append(b.e.Variants, v)
	return b
}
