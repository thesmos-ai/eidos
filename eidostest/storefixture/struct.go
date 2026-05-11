// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
)

// StructBuilder configures a [node.Struct] as part of a [Builder]'s
// accumulating package. The sub-builder is created by [Builder.Struct]
// and handed to that method's callback; the underlying struct is
// appended to the package after the callback returns.
//
// Field, method, and embed declarations within the callback see their
// owner back-pointers wired automatically, matching the shape a real
// frontend produces.
type StructBuilder struct {
	s       *node.Struct
	pkgPath string
}

// Node returns the underlying [node.Struct]. Use this accessor to set
// typed metadata or to assert against the node directly in tests.
func (b *StructBuilder) Node() *node.Struct { return b.s }

// Pos overrides the struct's source position. By default a struct
// constructed via the fixture carries a zero [position.Pos], which is
// the convention for synthetic test nodes that do not originate from
// a parsed source file.
func (b *StructBuilder) Pos(p position.Pos) *StructBuilder {
	b.s.SourcePos = p
	return b
}

// Docs appends doc-comment lines preserved verbatim. The lines are
// the raw textual content of the comment without `//` or `/* */`
// markers, matching what a frontend would record.
func (b *StructBuilder) Docs(lines ...string) *StructBuilder {
	b.s.DocLines = append(b.s.DocLines, lines...)
	return b
}

// Directive attaches d to the struct's directive list.
func (b *StructBuilder) Directive(d *directive.Directive) *StructBuilder {
	b.s.DirectiveList = append(b.s.DirectiveList, d)
	return b
}

// TypeParam declares a generic type parameter on the struct. Pass nil
// for constraint to omit it; a real frontend stamps an `any`-shaped
// ref when source omits the constraint, but the fixture does not
// synthesise this — tests should pass the constraint they care about.
func (b *StructBuilder) TypeParam(name string, constraint *node.TypeRef) *StructBuilder {
	b.s.TypeParams = append(b.s.TypeParams, &node.TypeParam{
		Name:       name,
		Constraint: constraint,
		Owner:      b.s,
	})
	return b
}

// Embed records an embedded type on the struct. Embedded types are
// distinct from named fields; consumers that want a unified view
// iterate both [node.Struct.Fields] and [node.Struct.Embeds].
func (b *StructBuilder) Embed(t *node.TypeRef) *StructBuilder {
	b.s.Embeds = append(b.s.Embeds, &node.Embed{Type: t, Owner: b.s})
	return b
}

// Field declares a named field on the struct and runs fn (when
// non-nil) against a [FieldBuilder] to configure it. The field's
// Owner back-pointer is set automatically.
func (b *StructBuilder) Field(name string, t *node.TypeRef, fn func(*FieldBuilder)) *StructBuilder {
	f := &node.Field{Name: name, Type: t, Owner: b.s}
	fb := &FieldBuilder{f: f}
	if fn != nil {
		fn(fb)
	}
	b.s.Fields = append(b.s.Fields, f)
	return b
}

// Method declares a method on the struct and runs fn (when non-nil)
// against a [MethodBuilder] to configure it. The method's Owner
// back-pointer is set to the enclosing struct; the receiver type
// defaults to a pointer-to-named ref for the struct (`*Pkg.Name`),
// matching the most common Go method-receiver shape. Override via
// [MethodBuilder.Receiver] when the test cares about value receivers.
func (b *StructBuilder) Method(name string, fn func(*MethodBuilder)) *StructBuilder {
	m := &node.Method{
		Name:     name,
		Receiver: Pointer(PkgNamed(b.pkgPath, b.s.Name)),
		Owner:    b.s,
	}
	mb := &MethodBuilder{m: m}
	if fn != nil {
		fn(mb)
	}
	b.s.Methods = append(b.s.Methods, m)
	return b
}
