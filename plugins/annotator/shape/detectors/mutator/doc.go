// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package mutator recognises the mutator shape — a callable that
// takes exactly one non-context value and returns nothing,
// performing an in-place or side-effecting mutation.
//
// The recognised Go signatures are:
//
//	func (r *Repo) Set(ctx context.Context, v V)
//	func (r *Repo) Set(v V)
//	func (r *Repo) Set(ctx context.Context, v *V) // pointer-receiver form
//
// A positive detection stamps:
//
//	shape            = "mutator"
//	shape.value_type = qualified type of `v` (or `*V`'s element)
package mutator
