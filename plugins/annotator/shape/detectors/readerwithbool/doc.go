// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package readerwithbool recognises a reader using a bool as the
// found / not-found discriminant — a callable that takes one key
// and returns a value alongside a sentinel `bool` (Go's
// map-lookup idiom).
//
// The recognised Go signature is:
//
//	func (r *Repo) Find(ctx context.Context, key K) (V, bool)
//
// A positive detection stamps:
//
//	shape            = "readerwithbool"
//	shape.key_type   = qualified type of `key`
//	shape.value_type = qualified type of `V`
package readerwithbool
