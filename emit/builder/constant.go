// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// ConstantBuilder configures an [emit.Constant] as part of a
// [PackageBuilder]'s accumulating package. Spawned by
// [PackageBuilder.Constant].
type ConstantBuilder struct {
	ctx *Context
	c   *emit.Constant
}

// Constant appends a package-level `const` declaration. typ may be
// nil when the constant's type is inferred from value. fn (which
// may be nil) configures docs / directives / position.
func (b *PackageBuilder) Constant(
	name string,
	typ emit.Ref,
	value *emit.Expr,
	fn func(*ConstantBuilder),
) *PackageBuilder {
	c := &emit.Constant{
		BaseEmit: emit.BaseEmit{SetByName: b.ctx.SetBy()},
		Name:     name,
		Package:  b.pkg.Path,
		Type:     typ,
		Value:    value,
		Target:   b.ctx.target,
	}
	cb := &ConstantBuilder{ctx: b.ctx, c: c}
	if fn != nil {
		fn(cb)
	}
	b.pkg.Constants = append(b.pkg.Constants, c)
	return b
}

// Node returns the underlying [emit.Constant].
func (b *ConstantBuilder) Node() *emit.Constant { return b.c }

// Target overrides the constant's [emit.Target].
func (b *ConstantBuilder) Target(t emit.Target) *ConstantBuilder {
	b.c.Target = t
	return b
}

// Pos overrides the constant's source position.
func (b *ConstantBuilder) Pos(p position.Pos) *ConstantBuilder {
	b.c.SourcePos = p
	return b
}

// Docs appends doc-comment lines above the constant declaration.
func (b *ConstantBuilder) Docs(lines ...string) *ConstantBuilder {
	b.c.DocLines = append(b.c.DocLines, lines...)
	return b
}

// Directive attaches d to the constant's directive list.
func (b *ConstantBuilder) Directive(d *directive.Directive) *ConstantBuilder {
	b.c.DirectiveList = append(b.c.DirectiveList, d)
	return b
}
