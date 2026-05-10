// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
)

// Node is the common interface every concrete emit kind satisfies.
//
// The shape mirrors [node.Node] for the foundation methods (Kind,
// Pos, Docs, Directives, Meta) and adds [Origin] — the back-link to
// the source [node.Node] this emit value was derived from, or nil
// for purely-generated artifacts (synthesised by a plugin without a
// source counterpart).
type Node interface {
	// Kind returns the [directive.Kind] discriminator for this emit
	// value (see the Kind* constants at package scope).
	Kind() directive.Kind
	// Pos returns the source position this value reflects. For
	// derived values it usually inherits from [Origin]; for
	// purely-generated values it is [position.Synthetic].
	Pos() position.Pos
	// Docs returns the documentation comment lines preserved or
	// freshly authored for this emit value.
	Docs() []string
	// Directives returns the directives carried on this emit value
	// (typically copied from the originating source node).
	Directives() []*directive.Directive
	// Directive returns the first directive with the given name,
	// or nil when none is attached.
	Directive(name directive.Name) *directive.Directive
	// HasDirective reports whether at least one directive with the
	// given name is attached.
	HasDirective(name directive.Name) bool
	// Meta returns the metadata bag for this emit value, allocating
	// one on first access.
	Meta() *meta.Bag
	// Origin returns the source node this emit value was derived
	// from, or nil for purely-generated artifacts. Templates and
	// generators use Origin to follow back to the source for
	// position, doc comments, and source-level metadata.
	Origin() node.Node
}
