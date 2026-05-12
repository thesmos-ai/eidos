// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// TypeParamBuilder configures an [emit.TypeParam] as part of a
// host declaration's generic-parameter list. Spawned by
// [StructBuilder.TypeParam], [InterfaceBuilder.TypeParam],
// [FunctionBuilder.TypeParam], [MethodBuilder.TypeParam], or
// [AliasBuilder.TypeParam]; the type parameter's Owner is wired by
// the spawning builder.
type TypeParamBuilder struct {
	ctx *Context
	p   *emit.TypeParam
}

// Node returns the underlying [emit.TypeParam].
func (b *TypeParamBuilder) Node() *emit.TypeParam { return b.p }

// Pos overrides the type parameter's source position.
func (b *TypeParamBuilder) Pos(p position.Pos) *TypeParamBuilder {
	b.p.SourcePos = p
	return b
}

// Docs appends doc-comment lines associated with the type parameter.
// Most renderers don't surface per-type-param documentation; the
// accessor is provided for symmetry with the other sub-builders and
// for plugins that thread directive lines.
func (b *TypeParamBuilder) Docs(lines ...string) *TypeParamBuilder {
	b.p.DocLines = append(b.p.DocLines, lines...)
	return b
}

// Directive attaches d to the type parameter's directive list.
func (b *TypeParamBuilder) Directive(d *directive.Directive) *TypeParamBuilder {
	b.p.DirectiveList = append(b.p.DirectiveList, d)
	return b
}

// Constraint overrides the type parameter's [emit.Constraint]. Use
// when the constraint isn't known at TypeParam call time, or to
// progressively populate a constraint via callbacks.
func (b *TypeParamBuilder) Constraint(c *emit.Constraint) *TypeParamBuilder {
	b.p.Constraint = c
	return b
}
