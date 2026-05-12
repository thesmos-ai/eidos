// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package domain holds the core entity types referenced by every
// other package in the multipkg fixture. The generator-targeted
// types live here so cross-package emit, generic instantiation,
// and same-package elision all converge on one well-known origin.
package domain

import "time"

// ID is a named type-definition layered over string. Aliases and
// type-definitions are different node-kinds; using one here
// exercises the type-definition path through refconv and renderType
// when storage's generic Repository instantiates over `domain.ID`.
type ID string

// User is one of the headline entities. Carries repo + builder
// directives so the storage-side mock and the in-package builder
// both emit alongside the source.
//
// +gen:repo
// +gen:builder
type User struct {
	ID       ID
	Name     string
	Email    string
	Created  time.Time
	Tags     []string
	Metadata map[string]string
}

// Order references User across the same package — testing
// same-package field-type elision when User and Order are both
// emitted/referenced from generated files in this package.
//
// +gen:repo
// +gen:builder
// +gen:register
type Order struct {
	ID         ID
	BuyerID    ID
	Buyer      *User
	Items      []OrderItem
	PlacedAt   time.Time
	TotalCents int64
}

// OrderItem isn't directly generator-targeted but participates in
// Order's composition. The plain (no-directive) sibling exercises
// the case where a referenced type is *defined* in the same package
// but is not itself emit-targeted — the builder for Order must
// reference OrderItem bare (no qualifier) without a self-import.
type OrderItem struct {
	ProductID ID
	Quantity  int
	Notes     string
}

// Product carries the third generator triple — repo + builder +
// register — so the multi-generator file-composition path is
// exercised for a non-User, non-Order entity too. The
// `+gen:out=product_codegen.go` directive renames the rendered
// builder file from the conventional `product_builder.go` to
// demonstrate the [pipeline.OutDirective] override path.
//
// +gen:repo
// +gen:builder
// +gen:register
// +gen:out=product_codegen.go
type Product struct {
	ID    ID
	Name  string
	Price int64
}
