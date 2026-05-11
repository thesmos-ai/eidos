// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// EnumVariant is one variant of an [Enum] in the emit tree. The
// Value is expressed as an [Expr] so backends render it consistently
// across languages.
type EnumVariant struct {
	BaseEmit

	// Name is the variant identifier.
	Name string `json:"name"`

	// Value is the variant's value expression.
	Value *Expr `json:"value,omitempty"`

	// Owner is the [Enum] this variant belongs to.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner *Enum `json:"-"`
}

// Kind returns [KindEnumVariant].
func (*EnumVariant) Kind() directive.Kind { return KindEnumVariant }
