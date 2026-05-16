// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package aggregator recognises the aggregator shape — a callable
// that takes nothing (besides the optional context) and returns
// a single value, optionally accompanied by an error.
//
// The recognised Go signatures are:
//
//	func (s *Stats) Count(ctx context.Context) (int, error)
//	func (s *Stats) Count(ctx context.Context) int
//	func (s *Stats) Count() int
//
// A positive detection stamps:
//
//	shape            = "aggregator"
//	shape.value_type = qualified type of the single value return
package aggregator
