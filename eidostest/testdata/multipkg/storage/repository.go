// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package storage holds the generic Repository surface. The
// `+gen:mock` directive on Repository[T] forces the mock generator
// to produce one mock per source interface. mockgen's default
// routes every mock into a `<srcPkg>_test` package with a
// `_mock_test.go` filename, so the rendered mock lands at
// `storage/repository_mock_test.go` declaring `package storage_test`
// — the Go toolchain compiles it only at test time and references
// back into the regular `storage` package qualify naturally because
// the test-package import identity differs.
package storage

import (
	"context"

	"example.com/multipkg/domain"
)

// Repository is the generic CRUD surface every entity-aware service
// consumes. The methods reference domain types (User, Order,
// Product) only through the type parameter T, so the rendered mock
// must thread the type-arg through every signature without losing
// the bracketed form.
//
// +gen:mock
type Repository[T any] interface {
	// Get fetches a single T by its identifier.
	Get(ctx context.Context, id domain.ID) (*T, error)

	// List returns a paginated set of T. The two-parameter generic
	// domain.Page exercises multi-arg instantiation rendering.
	List(ctx context.Context, cursor string) (domain.Page[T, string], error)

	// Save persists T and returns the canonical Result envelope.
	Save(ctx context.Context, value *T) domain.Result[*T]

	// Delete removes the T identified by id.
	Delete(ctx context.Context, id domain.ID) error
}

// Query is a generic predicate-driven query bundle. Its struct
// field types — domain.Filter[T] (a function type alias),
// domain.Box[T] (a single-parameter generic) — exercise the
// builder's ability to keep type arguments intact when emitting the
// `<Type>Builder` for a generic host.
//
// +gen:builder
type Query[T any] struct {
	Predicate domain.Filter[T]
	Pivot     domain.Box[T]
	Limit     int
}
