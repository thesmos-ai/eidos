// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package api wires the multipkg fixture's headline cross-package
// surface: it imports domain, both events packages (current and
// legacy — sharing a short name), storage, and internal/codec, so
// every alias-resolution path through the writer's ImportSet is
// exercised in one generated file.
package api

import (
	"context"

	"example.com/multipkg/domain"
	"example.com/multipkg/events"
	legacy "example.com/multipkg/legacy/events"
	"example.com/multipkg/storage"
)

// Handler is the API-side facade exposing the union of repository
// operations on User, Order, and Product, plus a handler-of-events
// surface that spans both events packages.
//
// +gen:mock
type Handler interface {
	// HandleUserEvent processes a current-shape user event. The
	// Event[T] generic instantiated over *domain.User exercises
	// cross-package generic-type rendering.
	HandleUserEvent(ctx context.Context, evt events.Event[*domain.User]) error

	// HandleLegacyEvent processes the legacy-shape event; the
	// `legacy` alias forces ImportSet collision resolution because
	// `legacy/events` shares its short name with the current
	// `events` package.
	HandleLegacyEvent(ctx context.Context, evt legacy.LegacyEvent) error

	// LookupUser pipes a Repository[*domain.User] through one call
	// site so storage's generic surface participates in the
	// rendered method signature.
	LookupUser(ctx context.Context, id domain.ID, repo storage.Repository[domain.User]) (*domain.User, error)
}

// Service bundles the dispatchers needed by the API surface. The
// generic Dispatcher[T] fields force the builder generator to
// thread type-args through field rendering and Build()'s composite
// literal.
//
// +gen:builder
type Service struct {
	UserDispatcher  *events.Dispatcher[*domain.User]
	OrderDispatcher *events.Dispatcher[*domain.Order]
	LegacyBus       legacy.LegacyDispatcher
	UserRepository  storage.Repository[domain.User]
	OrderRepository storage.Repository[domain.Order]
}
