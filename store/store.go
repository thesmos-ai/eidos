// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store

// Store is the indexed in-memory database the pipeline uses to share
// nodes and emit between phases. It exposes two views: a [NodeView]
// for source-side [node.Node] values populated by frontends, and an
// [EmitView] for [emit.Node] values populated by generators.
//
// The zero value is unusable; construct with [New].
//
// Methods on [Store] and the underlying views are safe for concurrent
// use. The pipeline's mutability contract serialises mutation phases
// in practice, but parallel annotators within a single bucket and
// parallel backend file rendering rely on the store's locking.
type Store struct {
	nodes *NodeView
	emit  *EmitView
}

// New returns an empty Store ready for use.
func New() *Store {
	return &Store{
		nodes: newNodeView(),
		emit:  newEmitView(),
	}
}

// Nodes returns the source-side view onto the store.
func (s *Store) Nodes() *NodeView { return s.nodes }

// Emit returns the output-side view onto the store.
func (s *Store) Emit() *EmitView { return s.emit }
