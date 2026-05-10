// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Interface is a method-set type emit. Methods declared inside an
// interface have a nil [Method.Receiver] — the receiver is implicit
// in the interface contract.
type Interface struct {
	BaseEmit

	// Name is the interface's identifier.
	Name string

	// Package is the package name the rendered file declares.
	// Empty for anonymous interface types.
	Package string

	// Methods are the interface's declared method signatures in
	// source order. Each Method has a nil Receiver.
	Methods []*Method

	// Embeds are the embedded interfaces (and union constraint
	// terms in Go's generic-type-set position) in source order.
	Embeds []*Embed

	// TypeParams are the interface's generic type parameters.
	TypeParams []*TypeParam

	// Target identifies where the backend writes this interface's
	// rendered output.
	Target Target

	slotMap
}

// Kind returns [KindInterface].
func (*Interface) Kind() directive.Kind { return KindInterface }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (i *Interface) QName() string {
	if i.Package == "" {
		return i.Name
	}
	return i.Package + "." + i.Name
}

// MethodsSlot returns the "methods" slot for cross-cutting method
// injection.
func (i *Interface) MethodsSlot() *Slot { return i.slot(i, "methods", KindMethod) }

// EmbedsSlot returns the "embeds" slot for cross-cutting embed
// injection.
func (i *Interface) EmbedsSlot() *Slot { return i.slot(i, "embeds", KindEmbed) }

// Slot returns the named slot, creating it lazily without an
// element-kind constraint.
func (i *Interface) Slot(name string) *Slot { return i.slot(i, name, "") }

// IsGeneric reports whether the interface declares generic type
// parameters.
func (i *Interface) IsGeneric() bool { return len(i.TypeParams) > 0 }

// MethodByName returns the method with the given name, or nil when
// no such method exists.
func (i *Interface) MethodByName(name string) *Method {
	for _, m := range i.Methods {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// MethodsWith returns methods matching pred in declaration order.
func (i *Interface) MethodsWith(pred func(*Method) bool) []*Method {
	out := make([]*Method, 0, len(i.Methods))
	for _, m := range i.Methods {
		if pred(m) {
			out = append(out, m)
		}
	}
	return out
}
