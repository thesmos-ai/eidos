// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/contract"
	"go.thesmos.sh/eidos/core/kind"
)

// Compile-time assertion that [*Struct] satisfies
// [contract.Owner] — fails to build if either accessor drifts.
var _ contract.Owner = (*Struct)(nil)

// Struct is a structured product type — Go's struct, Rust's struct,
// TypeScript's class, and so on at the model level. Embedded types
// surface as separate [Embed] nodes alongside the named Fields.
type Struct struct {
	BaseNode

	// Name is the struct's identifier.
	Name string `json:"name"`

	// Package is the source package path. Empty for anonymous
	// struct types declared inline.
	Package string `json:"package,omitempty"`

	// Fields are the named fields in source order.
	Fields []*Field `json:"fields,omitempty"`

	// Embeds are the embedded types in source order. Distinct from
	// Fields; consumers that want a unified view can iterate both.
	Embeds []*Embed `json:"embeds,omitempty"`

	// Methods declared on this struct (and on its pointer receiver
	// — frontends merge both sets).
	Methods []*Method `json:"methods,omitempty"`

	// TypeParams are the struct's generic type parameters.
	TypeParams []*TypeParam `json:"type_params,omitempty"`
}

// Kind returns [KindStruct].
func (*Struct) Kind() kind.Kind { return KindStruct }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (s *Struct) QName() string {
	if s.Package == "" {
		return s.Name
	}
	return s.Package + "." + s.Name
}

// OwnerName satisfies [contract.Owner]; returns the struct's
// bare identifier.
func (s *Struct) OwnerName() string { return s.Name }

// OwnerQName satisfies [contract.Owner]; synonym for
// [Struct.QName].
func (s *Struct) OwnerQName() string { return s.QName() }

// FieldByName returns the named field, or nil when no field with
// that name exists. Does not search [Embed] entries.
func (s *Struct) FieldByName(name string) *Field {
	for _, f := range s.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// FieldsWith returns fields matching pred in declaration order.
func (s *Struct) FieldsWith(pred func(*Field) bool) []*Field {
	out := make([]*Field, 0, len(s.Fields))
	for _, f := range s.Fields {
		if pred(f) {
			out = append(out, f)
		}
	}
	return out
}

// MethodByName returns the method with the given name, or nil when
// no such method exists.
func (s *Struct) MethodByName(name string) *Method {
	for _, m := range s.Methods {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// MethodsWith returns methods matching pred in declaration order.
func (s *Struct) MethodsWith(pred func(*Method) bool) []*Method {
	out := make([]*Method, 0, len(s.Methods))
	for _, m := range s.Methods {
		if pred(m) {
			out = append(out, m)
		}
	}
	return out
}

// IsGeneric reports whether the struct declares generic type
// parameters.
func (s *Struct) IsGeneric() bool { return len(s.TypeParams) > 0 }
