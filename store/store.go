// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"strings"

	"go.thesmos.sh/eidos/core/contract"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

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

// RewireMethodOwners repopulates [emit.Method.Owner] from
// [emit.Method.OwnerRef] for every top-level method whose Owner
// pointer is nil but whose OwnerRef carries a resolvable
// {Kind, QName} tuple. The pass is idempotent and the cache-replay
// hook the framework runs after deserialising a stored emit graph,
// reconstructing the "every routed top-level method has a
// meaningful Owner" invariant the layout and render passes rely on.
//
// Resolution dispatches on the OwnerRef Kind's namespace — bare
// kinds like "enum" resolve against the source-side store
// ([Store.Nodes]); kinds prefixed "emit." resolve against the
// emit-side store ([Store.Emit]). Unresolvable refs leave Owner
// nil (the framework surfaces this downstream as a routing error
// when the method reaches Layout); callers that want the resolve
// result re-walk the methods after the pass returns.
//
// Methods whose Owner is already populated are left untouched —
// the pass never clobbers a live pointer.
func (s *Store) RewireMethodOwners() {
	s.emit.Packages().Range(func(p *emit.Package) bool {
		for _, m := range p.Methods {
			if m.Owner != nil || m.OwnerRef.IsZero() {
				continue
			}
			if owner := s.resolveOwner(m.OwnerRef); owner != nil {
				m.Owner = owner
			}
		}
		return true
	})
}

// resolveOwner looks up the live [contract.Owner] referenced by
// ref. Returns nil when the bucket lookup fails — the caller
// leaves Owner nil and the surrounding pass surfaces the gap.
func (s *Store) resolveOwner(ref contract.OwnerRef) contract.Owner {
	if strings.HasPrefix(string(ref.Kind), "emit.") {
		return s.resolveEmitOwner(ref)
	}
	return s.resolveNodeOwner(ref)
}

// resolveNodeOwner dispatches the source-side resolve over the
// per-kind buckets on [Store.Nodes].
func (s *Store) resolveNodeOwner(ref contract.OwnerRef) contract.Owner {
	switch ref.Kind {
	case node.KindStruct:
		if v, ok := s.nodes.Structs().ByQName(ref.QName); ok {
			return v
		}
	case node.KindInterface:
		if v, ok := s.nodes.Interfaces().ByQName(ref.QName); ok {
			return v
		}
	case node.KindEnum:
		if v, ok := s.nodes.Enums().ByQName(ref.QName); ok {
			return v
		}
	case node.KindAlias:
		if v, ok := s.nodes.Aliases().ByQName(ref.QName); ok {
			return v
		}
	}
	return nil
}

// resolveEmitOwner dispatches the emit-side resolve over the
// per-kind buckets on [Store.Emit].
func (s *Store) resolveEmitOwner(ref contract.OwnerRef) contract.Owner {
	switch ref.Kind {
	case emit.KindStruct:
		if v, ok := s.emit.Structs().ByQName(ref.QName); ok {
			return v
		}
	case emit.KindInterface:
		if v, ok := s.emit.Interfaces().ByQName(ref.QName); ok {
			return v
		}
	case emit.KindAlias:
		if v, ok := s.emit.Aliases().ByQName(ref.QName); ok {
			return v
		}
	}
	return nil
}
