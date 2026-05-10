// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// TypeParam is one generic type parameter of a [Struct], [Interface],
// [Function], [Method], or [Alias]. The Constraint is the union or
// interface type the parameter must satisfy (Go's `T any` or
// `T comparable`).
type TypeParam struct {
	BaseNode

	// Name is the parameter identifier (e.g. "T", "K", "V").
	Name string

	// Constraint is the type that bounds the parameter. nil
	// indicates no explicit constraint (rare; usually the frontend
	// stamps a synthetic "any" reference).
	Constraint *TypeRef

	// Owner is the declaration the type parameter belongs to.
	// Populated by the constructing frontend.
	Owner Node
}

// Kind returns [KindTypeParam].
func (*TypeParam) Kind() directive.Kind { return KindTypeParam }

// IsConstrained reports whether the parameter declares an explicit
// constraint (Constraint != nil). Frontends typically populate even
// "unconstrained" parameters with a synthetic "any" constraint, so
// this returning false in practice signals incomplete frontend output.
func (p *TypeParam) IsConstrained() bool { return p.Constraint != nil }
