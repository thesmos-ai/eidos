// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package lookup recognises the lookup shape — a callable that
// takes one key and returns a value, its accompanying metadata,
// and a found / not-found bool sentinel.
//
// The recognised Go signature is:
//
//	func (r *Repo) Lookup(ctx context.Context, key K) (V, Meta, bool)
//
// A positive detection stamps:
//
//	shape                  = "lookup"
//	shape.key_type         = qualified type of `key`
//	shape.value_type       = qualified type of V
//	shape.lookup.meta_type = qualified type of Meta
package lookup
