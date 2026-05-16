// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/contract"
	"go.thesmos.sh/eidos/core/kind"
)

// Compile-time assertion that [*Alias] satisfies
// [contract.Owner] — fails to build if either accessor drifts.
var _ contract.Owner = (*Alias)(nil)

// Alias is a type alias or type definition — Go's `type X = Y` and
// `type X Y` forms. [Alias.IsAlias] distinguishes the two: true for
// the alias form, false for the definition form (which creates a new
// named type with the same underlying representation).
type Alias struct {
	BaseNode

	// Name is the alias identifier.
	Name string `json:"name"`

	// Package is the source package path.
	Package string `json:"package,omitempty"`

	// Target is the type being aliased / re-defined.
	Target *TypeRef `json:"target,omitempty"`

	// IsAlias is true for `type X = Y` (type alias) and false for
	// `type X Y` (new named type).
	IsAlias bool `json:"is_alias,omitempty"`

	// TypeParams are the alias's generic type parameters.
	TypeParams []*TypeParam `json:"type_params,omitempty"`

	// Methods declared on this named type. Methods on Go's `type X
	// Y` form attach here (Go allows methods on named types whose
	// underlying type is anything — basic, slice, map, channel,
	// etc.). True aliases (`type X = Y`) cannot carry methods of
	// their own, so this slice is empty for [Alias.IsAlias] true.
	Methods []*Method `json:"methods,omitempty"`
}

// Kind returns [KindAlias].
func (*Alias) Kind() kind.Kind { return KindAlias }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (a *Alias) QName() string {
	if a.Package == "" {
		return a.Name
	}
	return a.Package + "." + a.Name
}

// OwnerName satisfies [contract.Owner]; returns the alias's bare
// identifier.
func (a *Alias) OwnerName() string { return a.Name }

// OwnerQName satisfies [contract.Owner]; synonym for [Alias.QName].
func (a *Alias) OwnerQName() string { return a.QName() }

// IsGeneric reports whether the alias declares generic type
// parameters.
func (a *Alias) IsGeneric() bool { return len(a.TypeParams) > 0 }

// MethodByName returns the method named name, or nil when no such
// method exists.
func (a *Alias) MethodByName(name string) *Method {
	for _, m := range a.Methods {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// MethodsWith returns methods matching pred in declaration order.
func (a *Alias) MethodsWith(pred func(*Method) bool) []*Method {
	out := make([]*Method, 0, len(a.Methods))
	for _, m := range a.Methods {
		if pred(m) {
			out = append(out, m)
		}
	}
	return out
}
