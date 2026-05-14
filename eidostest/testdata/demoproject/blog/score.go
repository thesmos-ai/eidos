// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package blog

// Numeric is the type-set constraint accepted by the Score generic.
// Its `~int | ~float64` union exercises the backend's union-shape
// rendering path within a TypeParam constraint position.
type Numeric interface {
	~int | ~float64
}

// Score is a generic numeric envelope. Generators that emit code
// referencing Score must render the type parameter and its
// constraint correctly — the constraint resolves to a union shape.
type Score[T Numeric] struct {
	// Value is the wrapped numeric score.
	Value T
}
