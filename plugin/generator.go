// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

import (
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/store"
)

// Generator is the role for plugins that produce emit entities.
// Generators run in the generator phase, ordered across priority
// buckets and topo-sorted within a bucket. They read both
// Store.Nodes() (frozen at this point) and Store.Emit() (containing
// entities produced by earlier generators) and add new emit through
// Store.Emit().AddPackage and the slot APIs on existing emit hosts.
//
// Generate receives a [GeneratorContext]; per-emit issues attach to
// ctx.Diag and fatal failures return a non-nil error.
type Generator interface {
	Plugin

	// Generate runs the plugin's emit-production pass against
	// ctx.Store.
	Generate(ctx *GeneratorContext) error
}

// GeneratorContext is the per-run context handed to every
// [Generator]. The pipeline constructs one per generator with the
// shared store, the per-plugin read-tracking [store.Reader], and the
// shared diagnostic sink.
type GeneratorContext struct {
	// Store is the shared in-memory database. Generators read from
	// Store.Nodes() (frozen) and Store.Emit() (mutable during this
	// phase) and write new emit through Store.Emit().
	Store *store.Store

	// Reader is the per-plugin read-tracking handle. Generators
	// query through Reader so the captured reads contribute to the
	// plugin's cache key.
	Reader *store.Reader

	// Diag is the diagnostic sink shared with every plugin in the
	// run.
	Diag *diag.Sink
}
