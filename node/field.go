// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// Field is one named field of a [Struct] or of an inline anonymous
// struct type expression carried via [TypeRef] with kind
// [TypeRefAnonStruct]. Anonymous embedded fields use [Embed] nodes
// on the owning kind, not Field.
type Field struct {
	BaseNode

	// Name is the field identifier. Always non-empty for true
	// fields; embedded types use [Embed] instead.
	Name string

	// Type is the field's declared type.
	Type *TypeRef

	// Tag is the raw struct tag string (Go's backtick-quoted form),
	// without the enclosing backticks. Empty when no tag is declared.
	Tag string

	// Owner is the host that declares this field. For a [Struct]
	// field Owner is the *[Struct]; for a field inside an anonymous
	// struct type at a use site Owner is the enclosing *[TypeRef].
	// Populated by the constructing frontend.
	Owner Node
}

// Kind returns [KindField].
func (*Field) Kind() directive.Kind { return KindField }

// HasTag reports whether the field carries a struct tag.
func (f *Field) HasTag() bool { return f.Tag != "" }
