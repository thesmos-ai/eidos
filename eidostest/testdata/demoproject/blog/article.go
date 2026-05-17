// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package blog is the primary fixture package consumed by every
// demonstration plugin. Its declarations carry the canonical
// `+gen:` directives the plugins target and cover the rendering
// surface (composites, generics, enums, named returns, embedded
// interfaces, cross-package imports) the backend learned to handle.
//
// +gen:sentinel
package blog

import (
	"example.com/demoproject/extras"
)

// Article is the headline fixture entity — annotated for repository,
// builder, and registry generation simultaneously so the multi-
// generator file-composition path is exercised end-to-end against
// one canonical type.
//
// +gen:repo
// +gen:builder
// +gen:register
type Article struct {
	// ID identifies the article in storage. The cross-package UUID
	// type forces the backend to resolve an external import alias.
	ID extras.UUID

	// Title is the article headline.
	Title string

	// Body holds the article body text.
	Body string

	// Status is the article's publication state.
	Status Status

	// Tags is a free-form keyword list; the slice composite type
	// exercises the slice path in renderType.
	Tags []string

	// Meta carries arbitrary string metadata; the map composite type
	// exercises the map path in renderType.
	Meta map[string]string

	// Author back-references the User that wrote the article; the
	// pointer composite type exercises the pointer path in renderType.
	Author *User
}

// Validate reports whether the Article carries enough data to be
// persisted. The named return slot exercises the named-returns
// rendering case in the backend's renderReturns helper.
func (a *Article) Validate() (err error) {
	if a.Title == "" {
		return errBlankTitle
	}
	return nil
}

// String returns a human-readable summary of the Article. The
// anonymous-return slot complements Validate's named-return shape
// so both renderReturns cases are exercised by methods on the same
// host.
func (a *Article) String() string {
	return a.Title
}
