// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/contract"
	"go.thesmos.sh/eidos/core/kind"
)

// Compile-time assertion that [*Interface] satisfies
// [contract.Owner] — fails to build if either accessor drifts.
var _ contract.Owner = (*Interface)(nil)

// Interface is a method-set type — Go's interface, Rust's trait, and
// similar abstractions at the model level. Embedded interfaces surface
// as [Embed] nodes; explicitly-declared methods surface as Methods
// (with nil Receiver).
type Interface struct {
	BaseNode

	// Name is the interface's identifier.
	Name string `json:"name"`

	// Package is the source package path. Empty for anonymous
	// interface types declared inline.
	Package string `json:"package,omitempty"`

	// Methods are the interface's declared method signatures in
	// source order. Each Method has a nil Receiver — the receiver
	// is implicit in the interface.
	Methods []*Method `json:"methods,omitempty"`

	// Embeds are the embedded interfaces (and union constraint
	// terms in Go's generic-type-set position) in source order.
	Embeds []*Embed `json:"embeds,omitempty"`

	// TypeParams are the interface's generic type parameters.
	TypeParams []*TypeParam `json:"type_params,omitempty"`
}

// Kind returns [KindInterface].
func (*Interface) Kind() kind.Kind { return KindInterface }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (i *Interface) QName() string {
	if i.Package == "" {
		return i.Name
	}
	return i.Package + "." + i.Name
}

// OwnerName satisfies [contract.Owner]; returns the interface's
// bare identifier.
func (i *Interface) OwnerName() string { return i.Name }

// OwnerQName satisfies [contract.Owner]; synonym for
// [Interface.QName].
func (i *Interface) OwnerQName() string { return i.QName() }

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

// IsGeneric reports whether the interface declares generic type
// parameters.
func (i *Interface) IsGeneric() bool { return len(i.TypeParams) > 0 }
