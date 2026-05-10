// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Alias is a type alias or type definition emit — `type X = Y` and
// `type X Y` forms. [Alias.IsAlias] distinguishes the two: true for
// the alias form, false for the definition form (which creates a
// new named type with the same underlying representation).
type Alias struct {
	BaseEmit

	// Name is the alias identifier.
	Name string

	// Package is the package name the rendered file declares.
	Package string

	// Target is the type being aliased / re-defined.
	Target Ref

	// IsAlias is true for `type X = Y` (alias), false for
	// `type X Y` (new named type).
	IsAlias bool

	// TypeParams are the alias's generic type parameters.
	TypeParams []*TypeParam

	// File identifies where the backend writes this alias's
	// rendered output. Named "File" rather than "Target" to avoid
	// colliding with the [Alias.Target] type reference.
	File Target
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
