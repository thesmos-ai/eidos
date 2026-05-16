// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package voidlifecycle recognises the void-lifecycle shape — a
// callable that takes nothing, returns nothing, and exists only
// for its side-effects (`func ()` in Go).
//
// A positive detection stamps the structural shape on the
// callable's meta bag (via the umbrella shape plugin):
//
//	shape = "voidlifecycle"
//
// No key or value type is stamped — void-lifecycle callables
// have neither. Consumers register the [Detector] alongside
// other shape detectors when constructing the umbrella plugin.
package voidlifecycle
