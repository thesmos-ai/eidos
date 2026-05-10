// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// Constant is a package-level `const` declaration. Idiomatic Go
// "enum" patterns — a group of typed constants of the same type —
// are collected separately into [Enum] / [EnumVariant] by the
// frontend, but individual constants still appear here.
type Constant struct {
	BaseNode

	// Name is the constant identifier.
	Name string

	// Package is the source package path.
	Package string

	// Type is the declared type. May be nil when the type is
	// inferred from the value.
	Type *TypeRef

	// Value is the constant's value in verbatim source form.
	Value string
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
