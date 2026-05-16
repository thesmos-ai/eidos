// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package pointerreader recognises the pointer-reader shape — a
// callable that takes one key and returns a pointer-to-value
// (returning `nil` to signal absence rather than an error).
//
// The recognised Go signatures are:
//
//	func (c *Cache) Get(ctx context.Context, key K) *V
//	func (c *Cache) Get(key K) *V
//
// A positive detection stamps:
//
//	shape            = "pointerreader"
//	shape.key_type   = qualified type of `key`
//	shape.value_type = qualified type of `V` (the pointer's element)
//
// Register this detector before [readernoerror] so the pointer
// signature claims its own match instead of falling through to
// the more permissive single-return reader.
package pointerreader
