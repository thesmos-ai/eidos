// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package multiaggregator recognises an aggregator returning more
// than one value — a callable that takes only the optional
// context and returns N≥2 values followed by an error.
//
// The recognised Go signature is:
//
//	func (s *Stats) Pair(ctx context.Context) (V1, V2, error)
//
// A positive detection stamps:
//
//	shape                                = "multiaggregator"
//	shape.value_type                     = qualified type of V1
//	shape.multiaggregator.value_types    = qualified types of every non-error return
package multiaggregator
