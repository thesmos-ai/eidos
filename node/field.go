// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// Field is one field of a [Struct]. Anonymous embedded fields are
// represented as [Embed] nodes on the owning Struct, not as Field.
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

	// Owner is the [Struct] that declares this field. Populated by
	// the constructing frontend.
	Owner *Struct
}

// Kind returns [KindField].
func (*Field) Kind() directive.Kind { return KindField }

// HasTag reports whether the field carries a struct tag.
func (f *Field) HasTag() bool { return f.Tag != "" }
