// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// VariableBuilder configures an [emit.Variable] as part of a
// [PackageBuilder]'s accumulating package. Spawned by
// [PackageBuilder.Variable].
type VariableBuilder struct {
	ctx *Context
	v   *emit.Variable
}

// Variable appends a package-level `var` declaration. Either typ or
// init (or both) must be non-zero — Go's grammar requires at least
// one to infer or declare the variable's type. fn (which may be nil)
// configures docs / directives / position.
func (b *PackageBuilder) Variable(
	name string,
	typ emit.Ref,
	init *emit.Expr,
	fn func(*VariableBuilder),
) *PackageBuilder {
	v := &emit.Variable{
		BaseEmit: emit.BaseEmit{SetByName: b.ctx.SetBy()},
		Name:     name,
		Package:  b.pkg.Path,
		Type:     typ,
		Init:     init,
		Target:   b.ctx.target,
	}
	applyDefaultOrigin(b, &v.BaseEmit)
	vb := &VariableBuilder{ctx: b.ctx, v: v}
	if fn != nil {
		fn(vb)
	}
	b.pkg.Variables = append(b.pkg.Variables, v)
	return b
}

// Node returns the underlying [emit.Variable].
func (b *VariableBuilder) Node() *emit.Variable { return b.v }

// Target overrides the variable's [emit.Target].
func (b *VariableBuilder) Target(t emit.Target) *VariableBuilder {
	b.v.Target = t
	return b
}

// Pos overrides the variable's source position.
func (b *VariableBuilder) Pos(p position.Pos) *VariableBuilder {
	b.v.SourcePos = p
	return b
}

// Origin records the source [node.Node] this emit variable was
// derived from. Pass nil to clear an existing origin.
func (b *VariableBuilder) Origin(n node.Node) *VariableBuilder {
	b.v.OriginNode = n
	return b
}

// Docs appends doc-comment lines above the variable declaration.
func (b *VariableBuilder) Docs(lines ...string) *VariableBuilder {
	b.v.DocLines = append(b.v.DocLines, lines...)
	return b
}

// Directive attaches d to the variable's directive list.
func (b *VariableBuilder) Directive(d *directive.Directive) *VariableBuilder {
	b.v.DirectiveList = append(b.v.DirectiveList, d)
	return b
}
