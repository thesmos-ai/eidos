// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// FunctionBuilder configures an [emit.Function] as part of a
// [PackageBuilder]'s accumulating package. Spawned by
// [PackageBuilder.Function]; parameters, returns, and type
// parameters declared inside the callback receive their `Owner`
// back-pointer wired automatically.
type FunctionBuilder struct {
	ctx *Context
	f   *emit.Function
}

// Function appends a new top-level function named name to the
// package and runs fn against its [FunctionBuilder]. fn may be nil
// for an empty signature declaration. The constructed function
// carries Package = b.Node().Path and Target = ctx.Target().
func (b *PackageBuilder) Function(name string, fn func(*FunctionBuilder)) *PackageBuilder {
	f := &emit.Function{
		BaseEmit: emit.BaseEmit{SetByName: b.ctx.SetBy()},
		Name:     name,
		Package:  b.pkg.Path,
		Target:   b.ctx.target,
	}
	applyBuilderDefaults(b, &f.BaseEmit)
	fb := &FunctionBuilder{ctx: b.ctx, f: f}
	if fn != nil {
		fn(fb)
	}
	b.pkg.Functions = append(b.pkg.Functions, f)
	return b
}

// Node returns the underlying [emit.Function].
func (b *FunctionBuilder) Node() *emit.Function { return b.f }

// Target overrides the function's [emit.Target].
func (b *FunctionBuilder) Target(t emit.Target) *FunctionBuilder {
	b.f.Target = t
	return b
}

// Pos overrides the function's source position.
func (b *FunctionBuilder) Pos(p position.Pos) *FunctionBuilder {
	b.f.SourcePos = p
	return b
}

// Origin records the source [node.Node] this emit function was
// derived from. Pass nil to clear an existing origin.
func (b *FunctionBuilder) Origin(n node.Node) *FunctionBuilder {
	b.f.OriginNode = n
	return b
}

// Docs appends doc-comment lines above the function declaration.
func (b *FunctionBuilder) Docs(lines ...string) *FunctionBuilder {
	b.f.DocLines = append(b.f.DocLines, lines...)
	return b
}

// Directive attaches d to the function's directive list.
func (b *FunctionBuilder) Directive(d *directive.Directive) *FunctionBuilder {
	b.f.DirectiveList = append(b.f.DirectiveList, d)
	return b
}

// Param appends a positional parameter to the function. fn (which
// may be nil) configures Variadic flag and per-param metadata; the
// parameter's Owner is wired to the function automatically.
func (b *FunctionBuilder) Param(name string, t emit.Ref, fn func(*ParamBuilder)) *FunctionBuilder {
	p := &emit.Param{Name: name, Type: t, Owner: b.f}
	if fn != nil {
		fn(&ParamBuilder{ctx: b.ctx, p: p})
	}
	b.f.Params = append(b.f.Params, p)
	return b
}

// Return appends one [emit.Return] slot to the function. Pass an
// empty name for the anonymous-return form; named-return functions
// declare every slot with a non-empty name. Mixing the two surfaces
// at render time as [emit.ErrMixedNamedReturns].
func (b *FunctionBuilder) Return(t emit.Ref, name ...string) *FunctionBuilder {
	r := &emit.Return{Type: t}
	if len(name) > 0 {
		r.Name = name[0]
	}
	b.f.Returns = append(b.f.Returns, r)
	return b
}

// TypeParam appends a generic type parameter to the function. fn
// (which may be nil) configures position / docs / directives on
// the resulting [emit.TypeParam].
func (b *FunctionBuilder) TypeParam(
	name string,
	constraint *emit.Constraint,
	fn ...func(*TypeParamBuilder),
) *FunctionBuilder {
	p := &emit.TypeParam{Name: name, Constraint: constraint, Owner: b.f}
	if len(fn) > 0 && fn[0] != nil {
		fn[0](&TypeParamBuilder{ctx: b.ctx, p: p})
	}
	b.f.TypeParams = append(b.f.TypeParams, p)
	return b
}

// Body sets the function's statement body. Existing body statements
// are replaced; cross-cutting contributions go through the
// function's prebody / postbody slots rather than this setter.
func (b *FunctionBuilder) Body(stmts ...*emit.Stmt) *FunctionBuilder {
	b.f.Body = stmts
	return b
}
