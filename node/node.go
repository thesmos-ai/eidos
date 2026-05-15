// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/kind"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
)

// Node is the common interface every concrete node kind satisfies.
//
// The shape is intentionally narrow: a node identifies its kind and
// source position, exposes preserved documentation and parsed
// directives, and carries a metadata [Bag]. Kind-specific accessors
// (Methods, Fields, Params, …) live on the concrete types and are
// reached via type assertion or [Walk].
type Node interface {
	// Kind returns the [kind.Kind] discriminator for this node
	// (see the Kind* constants at package scope).
	Kind() kind.Kind
	// Pos returns where this node was declared in source. Synthetic
	// nodes use [position.Synthetic] markers.
	Pos() position.Pos
	// Docs returns the documentation comment lines preserved
	// verbatim from source (markers stripped). Empty for synthetic
	// nodes.
	Docs() []string
	// Directives returns every parsed +/-gen: directive attached to
	// this node in source order.
	Directives() []*directive.Directive
	// Directive returns the first directive with the given name, or
	// nil when none is attached.
	Directive(name directive.Name) *directive.Directive
	// HasDirective reports whether at least one directive with the
	// given name is attached.
	HasDirective(name directive.Name) bool
	// Meta returns the metadata bag for this node, allocating one
	// on first access. Plugins read and write through typed
	// [meta.Key] values.
	Meta() *meta.Bag
}
