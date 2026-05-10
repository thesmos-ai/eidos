// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Constant is a package-level `const` declaration in the emit tree.
// Like [Variable], the value is expressed as an [Expr] so backends
// render it in a uniform way.
type Constant struct {
	BaseEmit

	// Name is the constant identifier.
	Name string

	// Package is the package name the rendered file declares.
	Package string

	// Type is the declared type. May be nil when the type is
	// inferred from Value.
	Type Ref

	// Value is the constant's value expression.
	Value *Expr

	// Target identifies where the backend writes this constant.
	Target Target
}

// Kind returns [KindConstant].
func (*Constant) Kind() directive.Kind { return KindConstant }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (c *Constant) QName() string {
	if c.Package == "" {
		return c.Name
	}
	return c.Package + "." + c.Name
}

// HasDeclaredType reports whether the declaration carries an
// explicit type.
func (c *Constant) HasDeclaredType() bool { return c.Type != nil }
