// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// TypeParam is one generic type parameter of a [Struct], [Interface],
// [Function], [Method], or [Alias]. The Constraint is the structured
// bound the parameter must satisfy.
type TypeParam struct {
	BaseEmit

	// Name is the parameter identifier (e.g. "T", "K", "V").
	Name string

	// Constraint is the structured bound on the parameter. nil
	// indicates no explicit constraint (the implicit "any" of
	// languages with implicit bounds); see [Constraint.IsAny].
	Constraint *Constraint

	// Owner is the declaration this type parameter belongs to.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner Node `json:"-"`
}

// Kind returns [KindTypeParam].
func (*TypeParam) Kind() directive.Kind { return KindTypeParam }

// IsConstrained reports whether the parameter declares any explicit
// bound. A nil Constraint or one whose [Constraint.IsAny] returns
// true reads as unconstrained.
func (p *TypeParam) IsConstrained() bool {
	return p.Constraint != nil && !p.Constraint.IsAny()
}
