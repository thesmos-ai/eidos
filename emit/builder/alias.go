// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"errors"
	"fmt"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// ErrAliasMethodForbidden is recorded by [AliasBuilder.Method] when
// the surrounding alias is a true alias (`type X = Y`). Go's
// grammar forbids methods on true aliases; the resulting graph
// would render as invalid source. Callers compare with [errors.Is].
var ErrAliasMethodForbidden = errors.New("builder: methods are not allowed on a true alias (type X = Y)")

// AliasBuilder configures an [emit.Alias] as part of a
// [PackageBuilder]'s accumulating package. Spawned by
// [PackageBuilder.Alias] / [PackageBuilder.NamedType]; methods
// declared inside the callback receive their `Owner` back-pointer
// wired to the alias automatically.
type AliasBuilder struct {
	ctx    *Context
	parent *PackageBuilder
	a      *emit.Alias
}

// Alias appends a true type alias (`type X = Y`) named name and
// runs fn against its [AliasBuilder]. True aliases cannot carry
// methods — Go's language rule — so [AliasBuilder.Method] records
// [ErrAliasMethodForbidden] (surfaced via [PackageBuilder.Build]
// / [PackageBuilder.Err]) when called on a true alias. Use
// [PackageBuilder.NamedType] for the named-type form.
func (b *PackageBuilder) Alias(name string, target emit.Ref, fn func(*AliasBuilder)) *PackageBuilder {
	return b.appendAlias(name, target, true, fn)
}

// NamedType appends a named-type definition (`type X Y`) and runs
// fn against its [AliasBuilder]. Unlike true aliases, named types
// may carry methods.
func (b *PackageBuilder) NamedType(
	name string,
	underlying emit.Ref,
	fn func(*AliasBuilder),
) *PackageBuilder {
	return b.appendAlias(name, underlying, false, fn)
}

// appendAlias is the shared implementation of [PackageBuilder.Alias]
// and [PackageBuilder.NamedType] — the only difference between them
// is the IsAlias flag.
func (b *PackageBuilder) appendAlias(
	name string,
	target emit.Ref,
	isAlias bool,
	fn func(*AliasBuilder),
) *PackageBuilder {
	a := &emit.Alias{
		BaseEmit: emit.BaseEmit{SetByName: b.ctx.SetBy()},
		Name:     name,
		Package:  b.pkg.Path,
		Target:   target,
		IsAlias:  isAlias,
		File:     b.ctx.target,
	}
	applyDefaultOrigin(b, &a.BaseEmit)
	ab := &AliasBuilder{ctx: b.ctx, parent: b, a: a}
	if fn != nil {
		fn(ab)
	}
	b.pkg.Aliases = append(b.pkg.Aliases, a)
	return b
}

// Node returns the underlying [emit.Alias].
func (b *AliasBuilder) Node() *emit.Alias { return b.a }

// Origin records the source [node.Node] this emit alias was
// derived from. Pass nil to clear an existing origin.
func (b *AliasBuilder) Origin(n node.Node) *AliasBuilder {
	b.a.OriginNode = n
	return b
}

// Pos overrides the alias's source position.
func (b *AliasBuilder) Pos(p position.Pos) *AliasBuilder {
	b.a.SourcePos = p
	return b
}

// File overrides the alias's [emit.Target] (named `File` on
// [emit.Alias] to disambiguate from its type-target field).
func (b *AliasBuilder) File(t emit.Target) *AliasBuilder {
	b.a.File = t
	return b
}

// Docs appends doc-comment lines above the alias declaration.
func (b *AliasBuilder) Docs(lines ...string) *AliasBuilder {
	b.a.DocLines = append(b.a.DocLines, lines...)
	return b
}

// Directive attaches d to the alias's directive list.
func (b *AliasBuilder) Directive(d *directive.Directive) *AliasBuilder {
	b.a.DirectiveList = append(b.a.DirectiveList, d)
	return b
}

// TypeParam appends a generic type parameter. fn (which may be nil)
// configures position / docs / directives on the resulting
// [emit.TypeParam].
func (b *AliasBuilder) TypeParam(
	name string,
	constraint *emit.Constraint,
	fn ...func(*TypeParamBuilder),
) *AliasBuilder {
	p := &emit.TypeParam{Name: name, Constraint: constraint, Owner: b.a}
	if len(fn) > 0 && fn[0] != nil {
		fn[0](&TypeParamBuilder{ctx: b.ctx, p: p})
	}
	b.a.TypeParams = append(b.a.TypeParams, p)
	return b
}

// Method appends a method to this alias. Allowed only for named-type
// aliases (`type X Y`); calling Method on a true alias (`type X = Y`)
// records [ErrAliasMethodForbidden] on the parent
// [PackageBuilder]'s error list and discards the method — Go's
// grammar forbids methods on true aliases, and the resulting graph
// would render as invalid source. Recorded errors surface via
// [PackageBuilder.Build] / [PackageBuilder.Err].
func (b *AliasBuilder) Method(name string, fn func(*MethodBuilder)) *AliasBuilder {
	if b.a.IsAlias {
		b.parent.recordErr(fmt.Errorf("%w: alias %q method %q", ErrAliasMethodForbidden, b.a.Name, name))
		return b
	}
	m := &emit.Method{Name: name, Owner: b.a}
	if fn != nil {
		fn(&MethodBuilder{ctx: b.ctx, m: m})
	}
	b.a.Methods = append(b.a.Methods, m)
	return b
}
