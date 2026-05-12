// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package events (the *legacy* one — same short name as
// [example.com/multipkg/events]) preserves an older event shape any
// rolling migration still has to consume. The deliberate short-name
// collision with the current events package is the load-bearing
// fixture for the writer's ImportSet alias-collision discipline:
// importing both into one file forces one of them to render as
// `events2`.
package events

import "example.com/multipkg/domain"

// LegacyEvent is the older event shape. The historical record
// carried no type-parameter; the modern Event[T] superseded it.
//
// +gen:builder
type LegacyEvent struct {
	ID      domain.ID
	Topic   string
	Payload []byte
}

// LegacyDispatcher dispatches LegacyEvent values. The mock
// generator targets it; the rendered mock lands at
// `legacy/events/events_mock_test.go` declaring `package
// events_test` (mockgen's default external test-package
// routing) — exercising the cross-package short-name collision
// with the regular events package alongside the test-file
// build-tag convention.
//
// +gen:mock
type LegacyDispatcher interface {
	// Dispatch hands the LegacyEvent off to the legacy bus.
	Dispatch(evt LegacyEvent) error
}
