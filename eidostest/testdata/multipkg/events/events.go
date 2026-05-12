// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package events holds the current generic event surface. Its short
// name ("events") matches [example.com/multipkg/legacy/events] so
// any file importing both packages forces the writer's ImportSet to
// resolve the alias collision — the second-imported one renders as
// `events2` and references qualify through that suffixed alias.
package events

import (
	"time"

	"example.com/multipkg/domain"
)

// Event is the generic envelope every dispatched event carries. The
// type parameter T propagates through the payload field and the
// Handler interface below, exercising parameterised cross-package
// rendering.
type Event[T any] struct {
	ID        domain.ID
	Payload   T
	Timestamp time.Time
}

// Handler is the generic consumer surface. The mock generator
// targets it with the `+gen:mock` directive; the multipkg
// acceptance test wires mockgen with `test: true` so the rendered
// mock lands in `<src>_test.go` under the `events_test` package.
//
// +gen:mock
type Handler[T any] interface {
	// Handle processes one Event[T]. The return signal lets the
	// dispatcher track delivery confirmations.
	Handle(evt Event[T]) error
}

// Dispatcher orchestrates Handlers across multiple event types. The
// generic methods exercise the rendering of type-parameter receivers
// when buildergen emits the wrapper.
//
// +gen:builder
type Dispatcher[T any] struct {
	Handlers []Handler[T]
	Default  *Event[T]
}
