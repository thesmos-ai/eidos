// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package compositewriter recognises a writer that takes a key
// and a value — `(ctx?, K, V) error` — sometimes called a
// keyed-write or upsert-by-key shape.
//
// The recognised Go signature is:
//
//	func (r *Repo) Set(ctx context.Context, k K, v V) error
//
// A positive detection stamps:
//
//	shape            = "compositewriter"
//	shape.key_type   = qualified type of K
//	shape.value_type = qualified type of V
package compositewriter
