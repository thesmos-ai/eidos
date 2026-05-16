// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package store is the integration-test fixture for the testkit
// routing pattern. The `+gen:out storetest/` directive sends both
// the mock and its tests into a sibling `storetest/` package — the
// idiomatic Go test-helper layout downstream packages import from
// their own tests.
package store

import "context"

// Searcher is the fixture interface. The `+gen:out storetest/`
// directive routes the generated mock + tests into the sibling
// `storetest/` package; the `+gen:mock` directive opts the
// interface into mock generation.
//
// +gen:out storetest/
// +gen:mock
type Searcher interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key, value string)
}
