// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Field is one field of a [Struct]. Anonymous embedded fields are
// represented as [Embed] nodes on the owning Struct, not as Field.
//
// Field exposes a "tags" slot for cross-cutting generators to add
// struct-tag entries to a field without owning the field's owning
// generator (the validation, json, etc. plugins write into the slot).
type Field struct {
	BaseEmit

	// Name is the field identifier. Always non-empty for true
	// fields; embedded types use [Embed] instead.
	Name string

	// Type is the field's declared type.
	Type Ref

	// Tag is the raw struct-tag string (Go's backtick-quoted form),
	// without the enclosing backticks. Empty when no tag is
	// directly declared on the field — tag-slot contributions may
	// still add additional tag entries.
	Tag string

	// Owner is the [Struct] that declares this field.
	Owner *Struct

	slotMap
}

// Kind returns [KindField].
func (*Field) Kind() directive.Kind { return KindField }

// HasTag reports whether the field carries a directly-declared
// struct tag. Tag-slot contributions are not counted here; query
// [Field.Tags] for the cross-cutting view.
func (f *Field) HasTag() bool { return f.Tag != "" }

// Tags returns the "tags" slot used by cross-cutting generators to
// inject additional struct-tag entries. Items in the slot are
// language-/plugin-specific; backends render them alongside any
// directly-declared [Field.Tag].
func (f *Field) Tags() *Slot {
	return f.slot(f, "tags", "")
}

// SlotByName returns the named slot on the field, creating it
// lazily for ad-hoc cross-cutting contributions outside the
// standard "tags" slot.
func (f *Field) SlotByName(name string) *Slot {
	return f.slot(f, name, "")
}
