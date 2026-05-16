// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/contract"
	"go.thesmos.sh/eidos/node"
)

// Node is the common interface every concrete emit kind satisfies.
// It embeds [contract.Node] — the seven-method cross-graph
// foundation — and adds the two emit-specific accessors every
// emit value carries: [Origin], the back-link to the source
// node the emit value was derived from, and [SetBy], the plugin
// attribution stamped by the builder context.
//
// emit values can be passed to framework components that accept
// [contract.Node] (the routing layer, owner-resolve passes,
// cross-graph plugin queries) because of the embedding; emit-only
// templates and generators that need Origin / SetBy work against
// emit.Node directly.
type Node interface {
	contract.Node

	// Origin returns the source node this emit value was derived
	// from, or nil for purely-generated artifacts. Templates and
	// generators use Origin to follow back to the source for
	// position, doc comments, and source-level metadata.
	Origin() node.Node

	// SetBy returns the plugin identifier that produced this emit
	// value. Empty for entities constructed without a builder
	// context. Backends use it to compose per-target plugin
	// attribution (the rendered file's `Plugins:` header, the
	// run's manifest entry's Plugins list); the empty string is
	// treated as "unattributed".
	SetBy() string
}
