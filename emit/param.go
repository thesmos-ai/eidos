// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/kind"
)

// Param is one positional parameter of a [Function] or [Method].
//
// Variadic parameters carry Variadic = true; the parameter's Type
// is the slice element type, matching Go semantics
// ("...int" → Type is Builtin("int"), Variadic is true).
type Param struct {
	BaseEmit

	// Name is the parameter identifier. Empty for anonymous
	// parameters in function-type fields and interface signatures.
	Name string `json:"name,omitempty"`

	// Type is the parameter's declared type.
	Type Ref `json:"-"`

	// Variadic reports whether this parameter is variadic — must
	// be the last parameter in the parameter list.
	Variadic bool `json:"variadic,omitempty"`

	// Owner is the [Function] or [Method] this parameter belongs to.
	// Populated by the constructing generator.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner Node `json:"-"`
}

// Kind returns [KindParam].
func (*Param) Kind() kind.Kind { return KindParam }

// IsAnonymous reports whether the parameter has no name.
func (p *Param) IsAnonymous() bool { return p.Name == "" }
