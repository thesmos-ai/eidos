// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

// NodesOnly is the optional capability for [Generator] plugins that
// promise to read only the source-side store ([store.Store.Nodes])
// during their phase, never the emit graph populated by earlier
// generators. The pipeline uses the declaration to parallelise the
// generator phase: NodesOnly generators within the same priority
// bucket may run concurrently, while generators that depend on
// upstream emit serialise as usual.
//
// Plugins that read [store.Store.Emit] from earlier generators MUST
// NOT implement this interface — concurrent reads of an emit graph
// that another generator is mutating would race.
type NodesOnly interface {
	// NodesOnly reports whether the generator restricts its store
	// access to the source-side view. A constant return value is
	// expected (the declaration is a static contract, not a runtime
	// switch).
	NodesOnly() bool
}
