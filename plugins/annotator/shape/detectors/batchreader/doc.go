// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package batchreader recognises a batched reader — a callable
// that takes a variadic list of keys and returns the matching
// values as a slice, accompanied by an error.
//
// The recognised Go signature is:
//
//	func (r *Repo) GetAll(ctx context.Context, ids ...K) ([]V, error)
//
// A positive detection stamps:
//
//	shape            = "batchreader"
//	shape.key_type   = qualified type of K (the variadic element)
//	shape.value_type = qualified type of V (the slice element)
package batchreader
