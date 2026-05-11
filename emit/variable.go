// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Variable is a package-level `var` declaration in the emit tree.
// Initial values are expressed as an [Expr] so backends can render
// the initialiser correctly; raw verbatim text falls through via
// [NewRawExpr].
type Variable struct {
	BaseEmit

	// Name is the variable identifier.
	Name string `json:"name"`

	// Package is the package name the rendered file declares.
	Package string `json:"package,omitempty"`

	// Type is the declared type. May be nil when the type is
	// inferred from Init.
	Type Ref `json:"-"`

	// Init is the initialiser expression. nil for declarations
	// without an initialiser.
	Init *Expr `json:"init,omitempty"`

	// Target identifies where the backend writes this variable's
	// rendered output.
	Target Target `json:"target,omitzero"`
}

// Kind returns [KindVariable].
func (*Variable) Kind() directive.Kind { return KindVariable }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (v *Variable) QName() string {
	if v.Package == "" {
		return v.Name
	}
	return v.Package + "." + v.Name
}

// HasInit reports whether the declaration carries an initialiser
// expression.
func (v *Variable) HasInit() bool { return v.Init != nil }

// HasDeclaredType reports whether the declaration carries an
// explicit type.
func (v *Variable) HasDeclaredType() bool { return v.Type != nil }
