// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package predicate recognises the predicate shape — a callable
// that takes nothing and returns a single bare bool
// (`func () bool` in Go).
//
// A positive detection stamps the structural shape on the
// callable's meta bag (via the umbrella shape plugin):
//
//	shape = "predicate"
//
// No key or value type is stamped — the bool return is implied
// by the shape itself.
package predicate
