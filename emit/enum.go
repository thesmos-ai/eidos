// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Enum is the emit-level enum model — a named type plus a set of
// typed variants. The "variants" slot allows cross-cutting
// generators to inject additional variants without owning the enum.
type Enum struct {
	BaseEmit

	// Name is the enum's type identifier.
	Name string `json:"name"`

	// Package is the package name the rendered file declares.
	Package string `json:"package,omitempty"`

	// Underlying is the enum's underlying type.
	Underlying Ref `json:"-"`

	// Variants are the declared variants in source order.
	// Cross-cutting variant injection appends through
	// [Enum.VariantsSlot].
	Variants []*EnumVariant `json:"variants,omitempty"`

	// Target identifies where the backend writes this enum's
	// rendered output.
	Target Target `json:"target,omitzero"`

	slotMap
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

// VariantsSlot returns the "variants" slot for cross-cutting
// variant injection.
func (e *Enum) VariantsSlot() *Slot { return e.slot(e, "variants", KindEnumVariant) }

// Slot returns the named slot, creating it lazily.
func (e *Enum) Slot(name string) *Slot { return e.slot(e, name, "") }

// HasUnderlying reports whether the enum declares an underlying type.
func (e *Enum) HasUnderlying() bool { return e.Underlying != nil }

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
