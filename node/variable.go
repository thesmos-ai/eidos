// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/kind"
)

// Variable is a package-level `var` declaration. Initial-value
// information is preserved verbatim in [Variable.InitExpr] so
// downstream tools can render the original literal without
// re-evaluation.
type Variable struct {
	BaseNode

	// Name is the variable identifier.
	Name string `json:"name"`

	// Package is the source package path.
	Package string `json:"package,omitempty"`

	// Type is the declared type. May be nil for variables whose
	// type is inferred from InitExpr.
	Type *TypeRef `json:"type,omitempty"`

	// InitExpr is the verbatim initialiser expression text. Empty
	// for declarations without an initialiser.
	InitExpr string `json:"init_expr,omitempty"`
}

// Kind returns [KindVariable].
func (*Variable) Kind() kind.Kind { return KindVariable }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (v *Variable) QName() string {
	if v.Package == "" {
		return v.Name
	}
	return v.Package + "." + v.Name
}

// HasInitExpr reports whether the declaration carries an initialiser
// expression.
func (v *Variable) HasInitExpr() bool { return v.InitExpr != "" }

// HasDeclaredType reports whether the declaration carries an explicit
// type. Variables whose type is inferred from the initialiser have
// no declared type.
func (v *Variable) HasDeclaredType() bool { return v.Type != nil }
