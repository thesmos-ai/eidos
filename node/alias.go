// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// Alias is a type alias or type definition — Go's `type X = Y` and
// `type X Y` forms. [Alias.IsAlias] distinguishes the two: true for
// the alias form, false for the definition form (which creates a new
// named type with the same underlying representation).
type Alias struct {
	BaseNode

	// Name is the alias identifier.
	Name string

	// Package is the source package path.
	Package string

	// Target is the type being aliased / re-defined.
	Target *TypeRef

	// IsAlias is true for `type X = Y` (type alias) and false for
	// `type X Y` (new named type).
	IsAlias bool

	// TypeParams are the alias's generic type parameters.
	TypeParams []*TypeParam
}

// Kind returns [KindAlias].
func (*Alias) Kind() directive.Kind { return KindAlias }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (a *Alias) QName() string {
	if a.Package == "" {
		return a.Name
	}
	return a.Package + "." + a.Name
}

// IsGeneric reports whether the alias declares generic type
// parameters.
func (a *Alias) IsGeneric() bool { return len(a.TypeParams) > 0 }
