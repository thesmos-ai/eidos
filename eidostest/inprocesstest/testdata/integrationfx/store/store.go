// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package store is the integration-test fixture targeted by the
// mock + mocktest generator pair. The interface declares one
// representative method shape — a context.Context plus a builtin
// parameter, a builtin return alongside the error sentinel — so
// the rendered output exercises external-import resolution,
// named-return shaping, and the mocktest override closure all in
// one pass.
package store

import "context"

// Searcher is the fixture interface. The `+gen:mock` directive
// opts it into mock + mocktest generation.
//
// +gen:mock
type Searcher interface {
	// Get returns the value stored at key, or the empty string
	// when absent. The signature stays minimal on purpose — the
	// integration check is "does the rendered code compile",
	// not "does it model a realistic store".
	Get(ctx context.Context, key string) (string, error)

	// Put writes value at key. The void-return shape exercises
	// the dispatch body's "no trailing naked return" branch.
	Put(ctx context.Context, key, value string)
}
