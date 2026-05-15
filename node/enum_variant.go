// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/kind"
)

// EnumVariant is one variant of an idiomatic enum — a typed constant
// whose type matches the enclosing [Enum]'s underlying type.
//
// In Go, EnumVariants are individual const declarations grouped by
// shape detection or explicit grouping by the frontend. Other
// language frontends with first-class enums (Rust, TypeScript)
// produce EnumVariants directly from the language's enum syntax.
type EnumVariant struct {
	BaseNode

	// Name is the variant identifier.
	Name string `json:"name"`

	// Value is the variant's value in verbatim source form
	// (typically a literal integer or string).
	Value string `json:"value,omitempty"`

	// Owner is the [Enum] this variant belongs to. Populated by
	// the constructing frontend.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner *Enum `json:"-"`
}

// Kind returns [KindEnumVariant].
func (*EnumVariant) Kind() kind.Kind { return KindEnumVariant }
