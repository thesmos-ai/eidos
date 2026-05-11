// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// TypeParam is one generic type parameter of a [Struct], [Interface],
// [Function], [Method], or [Alias]. The Constraint is the structured
// bound the parameter must satisfy (Go's `T any`, `T comparable`,
// `T ~int | ~string`, or the combined `T Stringer; ~int`).
type TypeParam struct {
	BaseNode

	// Name is the parameter identifier (e.g. "T", "K", "V").
	Name string

	// Constraint is the structured bound on the parameter. nil
	// indicates no explicit constraint (the implicit `any`); see
	// [Constraint.IsAny].
	Constraint *Constraint

	// Owner is the declaration the type parameter belongs to.
	// Populated by the constructing frontend.
	Owner Node
}

// Kind returns [KindTypeParam].
func (*TypeParam) Kind() directive.Kind { return KindTypeParam }

// IsConstrained reports whether the parameter declares any explicit
// bound. A nil Constraint or one whose [Constraint.IsAny] returns
// true reads as unconstrained.
func (p *TypeParam) IsConstrained() bool {
	return p.Constraint != nil && !p.Constraint.IsAny()
}
