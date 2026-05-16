// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package poisonaccessor recognises the poison-accessor shape —
// a callable that takes nothing and reports success or failure
// solely via an error return (`func () error` in Go).
//
// A positive detection stamps the structural shape on the
// callable's meta bag (via the umbrella shape plugin):
//
//	shape = "poisonaccessor"
//
// No key or value type is stamped — poison accessors have
// neither.
package poisonaccessor
