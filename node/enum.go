// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// Enum is the language-agnostic enum model — a named type plus a set
// of typed variants of that underlying type.
//
// Frontends populate Enum via different routes per language:
//
//   - Go: shape detection over a group of typed constants
//     (`type Status int` + `const (StatusActive Status = iota; …)`).
//   - Rust / TypeScript / others: direct translation from
//     first-class enum / discriminated-union syntax.
//
// The model is intentionally simple — typed variants over an
// underlying type — and does not attempt to capture Rust-style
// payload variants. Frontends that need to model payload-carrying
// enums use a combination of [Enum] + plugin-specific metadata or
// represent them as interface + struct variants instead.
type Enum struct {
	BaseNode

	// Name is the enum's type identifier.
	Name string

	// Package is the source package path.
	Package string

	// Underlying is the enum's underlying type (Go's `type Status
	// int` would set this to a Named TypeRef for "int").
	Underlying *TypeRef

	// Variants are the declared variants in source order.
	Variants []*EnumVariant
}

// Kind returns [KindEnum].
func (*Enum) Kind() directive.Kind { return KindEnum }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (e *Enum) QName() string {
	if e.Package == "" {
		return e.Name
	}
	return e.Package + "." + e.Name
}

// VariantByName returns the variant with the given name, or nil
// when no such variant exists.
func (e *Enum) VariantByName(name string) *EnumVariant {
	for _, v := range e.Variants {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// VariantsWith returns variants matching pred in declaration order.
func (e *Enum) VariantsWith(pred func(*EnumVariant) bool) []*EnumVariant {
	out := make([]*EnumVariant, 0, len(e.Variants))
	for _, v := range e.Variants {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// HasUnderlying reports whether the enum declares an underlying type.
// Frontends that produce typeless enums leave this nil; downstream
// consumers can detect that and fall back to a default type.
func (e *Enum) HasUnderlying() bool { return e.Underlying != nil }
