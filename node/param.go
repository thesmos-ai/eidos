// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// Param is one positional parameter of a [Function] or [Method].
//
// Variadic parameters carry Variadic = true; the parameter's Type
// is the slice element type, matching Go's semantics ("..int" → Type
// is "int", Variadic is true).
type Param struct {
	BaseNode

	// Name is the parameter identifier. May be empty for anonymous
	// parameters in function signatures.
	Name string

	// Type is the parameter's declared type.
	Type *TypeRef

	// Variadic reports whether this parameter is variadic (must be
	// the last parameter in the parameter list).
	Variadic bool

	// Owner is the function or method this parameter belongs to.
	// Populated by the constructing frontend.
	Owner Node
}

// Kind returns [KindParam].
func (*Param) Kind() directive.Kind { return KindParam }

// IsAnonymous reports whether the parameter has no name. Common for
// interface method signatures and function-type fields.
func (p *Param) IsAnonymous() bool { return p.Name == "" }
