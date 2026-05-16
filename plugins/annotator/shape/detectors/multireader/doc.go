// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package multireader recognises a reader returning more than one
// value — a callable that takes one key and returns N≥2 values
// followed by an error.
//
// The recognised Go signature is:
//
//	func (r *Repo) Get(ctx context.Context, key K) (V1, V2, error)
//
// A positive detection stamps:
//
//	shape                          = "multireader"
//	shape.key_type                 = qualified type of `key`
//	shape.value_type               = qualified type of V1
//	shape.multireader.value_types  = qualified types of every non-error return
package multireader
