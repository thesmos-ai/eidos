// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
)

// WithDirective returns a predicate matching any [Node] that carries
// a directive with the given name.
//
// Use with the *With accessor families:
//
//	repos := pkg.Structs                                  // []*Struct
//	annotated := slices.Filter(repos, node.WithDirective("repo"))
//
// Predicates compose via [And], [Or], and [Not].
func WithDirective(name directive.Name) func(Node) bool {
	return func(n Node) bool { return n.HasDirective(name) }
}

// WithMeta returns a predicate matching any [Node] whose metadata
// bag has the named key set (defaults are not consulted; only
// explicit values).
func WithMeta(name string) func(Node) bool {
	return func(n Node) bool { return n.Meta().Has(name) }
}

// WithMetaKey is the typed sibling of [WithMeta] — it returns a
// predicate that matches when the key resolves to a value (any
// non-tombstoned value at any authority).
func WithMetaKey[T any](k meta.Key[T]) func(Node) bool {
	return func(n Node) bool { return k.Has(n.Meta()) }
}

// And returns a predicate that matches when every supplied predicate
// matches. Empty input returns a predicate that matches everything.
func And[T any](preds ...func(T) bool) func(T) bool {
	return func(v T) bool {
		for _, p := range preds {
			if !p(v) {
				return false
			}
		}
		return true
	}
}

// Or returns a predicate that matches when any supplied predicate
// matches. Empty input returns a predicate that matches nothing.
func Or[T any](preds ...func(T) bool) func(T) bool {
	return func(v T) bool {
		for _, p := range preds {
			if p(v) {
				return true
			}
		}
		return false
	}
}

// Not inverts pred.
func Not[T any](pred func(T) bool) func(T) bool {
	return func(v T) bool { return !pred(v) }
}
