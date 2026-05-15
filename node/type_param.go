// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/kind"
)

// TypeParam is one generic type parameter of a [Struct], [Interface],
// [Function], [Method], or [Alias]. The Constraint is the structured
// bound the parameter must satisfy (Go's `T any`, `T comparable`,
// `T ~int | ~string`, or the combined `T Stringer; ~int`).
type TypeParam struct {
	BaseNode

	// Name is the parameter identifier (e.g. "T", "K", "V").
	Name string `json:"name"`

	// Constraint is the structured bound on the parameter. nil
	// indicates no explicit constraint (the implicit `any`); see
	// [Constraint.IsAny].
	Constraint *Constraint `json:"constraint,omitempty"`

	// Owner is the declaration the type parameter belongs to.
	// Populated by the constructing frontend.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner Node `json:"-"`
}

// Kind returns [KindTypeParam].
func (*TypeParam) Kind() kind.Kind { return KindTypeParam }

// IsConstrained reports whether the parameter declares any explicit
// bound. A nil Constraint or one whose [Constraint.IsAny] returns
// true reads as unconstrained.
func (p *TypeParam) IsConstrained() bool {
	return p.Constraint != nil && !p.Constraint.IsAny()
}
