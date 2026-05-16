// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package readernoerror recognises a reader signature without an
// error return — a callable that takes one key and returns one
// value with no failure channel.
//
// The recognised Go signatures are:
//
//	func (c *Cache) Get(ctx context.Context, key K) V
//	func (c *Cache) Get(key K) V
//
// A positive detection stamps:
//
//	shape            = "readernoerror"
//	shape.key_type   = qualified type of `key`
//	shape.value_type = qualified type of `V`
//
// Register this detector *after* shapes that constrain the
// signature further ([pointerreader], …) so they claim their
// matches first; readernoerror is the fallback for the "key in,
// non-error value out" pattern.
package readernoerror
