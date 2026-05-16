// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package store is the integration-test fixture for the
// per-directive routing-keys path. The `+gen:mock` directive
// itself carries `out=` and `pkg=`, scoping the routing override
// to the mock plugin without a separate `+gen:out` line.
package store

import "context"

// Searcher is the fixture interface. The `+gen:mock out=...
// pkg=...` form bundles the routing override into the directive
// that triggers emission — the natural anchor.
//
// +gen:mock out=storetest/ pkg=storetest
type Searcher interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key, value string)
}
