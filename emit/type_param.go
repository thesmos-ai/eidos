// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// TypeParam is one generic type parameter of a [Struct], [Interface],
// [Function], [Method], or [Alias]. The Constraint is the union /
// interface ref the parameter must satisfy (Go's `T any` or
// `T comparable`).
type TypeParam struct {
	BaseEmit

	// Name is the parameter identifier (e.g. "T", "K", "V").
	Name string

	// Constraint is the type that bounds the parameter. nil
	// indicates no explicit constraint; generators that want a
	// universal bound typically stamp BuiltinRef("any").
	Constraint Ref

	// Owner is the declaration this type parameter belongs to.
	Owner Node
}

// Kind returns [KindTypeParam].
func (*TypeParam) Kind() directive.Kind { return KindTypeParam }

// IsConstrained reports whether the parameter declares an explicit
// constraint.
func (p *TypeParam) IsConstrained() bool { return p.Constraint != nil }
