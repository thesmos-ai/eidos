// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// ParamBuilder configures an [emit.Param] as part of a function /
// method parameter list. Spawned by [FunctionBuilder.Param] or
// [MethodBuilder.Param]; the param's Owner is wired by the spawning
// builder.
type ParamBuilder struct {
	ctx *Context
	p   *emit.Param
}

// Node returns the underlying [emit.Param].
func (b *ParamBuilder) Node() *emit.Param { return b.p }

// Pos overrides the parameter's source position.
func (b *ParamBuilder) Pos(p position.Pos) *ParamBuilder {
	b.p.SourcePos = p
	return b
}

// Docs appends doc-comment lines above the parameter.
func (b *ParamBuilder) Docs(lines ...string) *ParamBuilder {
	b.p.DocLines = append(b.p.DocLines, lines...)
	return b
}

// Directive attaches d to the parameter's directive list.
func (b *ParamBuilder) Directive(d *directive.Directive) *ParamBuilder {
	b.p.DirectiveList = append(b.p.DirectiveList, d)
	return b
}

// Variadic marks the parameter as variadic. The renderer rejects
// non-last variadic parameters, so callers should only mark the
// trailing parameter of a function or method.
func (b *ParamBuilder) Variadic() *ParamBuilder {
	b.p.Variadic = true
	return b
}
