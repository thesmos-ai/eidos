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
	Name string

	// Value is the variant's value expression.
	Value *Expr

	// Owner is the [Enum] this variant belongs to.
	Owner *Enum
}

// Kind returns [KindEnumVariant].
func (*EnumVariant) Kind() directive.Kind { return KindEnumVariant }
