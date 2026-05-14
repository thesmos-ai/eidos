// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
)

// InterfaceBuilder configures a [node.Interface] within a [Builder]'s
// accumulating package. Methods declared via [InterfaceBuilder.Method]
// have a nil [node.Method.Receiver] — the convention for interface
// methods, mirroring how a Go frontend records them.
type InterfaceBuilder struct {
	i       *node.Interface
	pkgPath string
}

// Node returns the underlying [node.Interface].
func (b *InterfaceBuilder) Node() *node.Interface { return b.i }

// Pos overrides the interface's source position.
func (b *InterfaceBuilder) Pos(p position.Pos) *InterfaceBuilder {
	b.i.SourcePos = p
	return b
}

// Docs appends doc-comment lines.
func (b *InterfaceBuilder) Docs(lines ...string) *InterfaceBuilder {
	b.i.DocLines = append(b.i.DocLines, lines...)
	return b
}

// Directive attaches d to the interface's directive list.
func (b *InterfaceBuilder) Directive(d *directive.Directive) *InterfaceBuilder {
	b.i.DirectiveList = append(b.i.DirectiveList, d)
	return b
}

// TypeParam declares a generic type parameter on the interface. Pass
// nil for an implicit "any" bound, or use [Constraint] for an
// explicit named-bound constraint.
func (b *InterfaceBuilder) TypeParam(name string, constraint *node.Constraint) *InterfaceBuilder {
	b.i.TypeParams = append(b.i.TypeParams, &node.TypeParam{
		Name:       name,
		Constraint: constraint,
		Owner:      b.i,
	})
	return b
}

// Embed records an embedded interface or type on the interface.
func (b *InterfaceBuilder) Embed(t *node.TypeRef) *InterfaceBuilder {
	b.i.Embeds = append(b.i.Embeds, &node.Embed{Type: t, Owner: b.i})
	return b
}

// Method declares a method on the interface. The method's Receiver
// stays nil (matching Go's interface-method shape); the Owner
// back-pointer is set to the enclosing interface.
func (b *InterfaceBuilder) Method(name string, fn func(*MethodBuilder)) *InterfaceBuilder {
	m := &node.Method{Name: name, Owner: b.i}
	mb := &MethodBuilder{m: m}
	if fn != nil {
		fn(mb)
	}
	b.i.Methods = append(b.i.Methods, m)
	return b
}
