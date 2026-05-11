// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
)

// directiveCarrier is the minimal contract a value must satisfy to
// be filtered by [WithDirective]. Both [node.Node] and [emit.Node]
// satisfy it, so the predicate works uniformly for source-side and
// emit-side queries.
type directiveCarrier interface {
	HasDirective(directive.Name) bool
}

// metaCarrier is the minimal contract a value must satisfy to be
// filtered by [WithMeta], [WithMetaKey], and [MetaEq]. The interface
// is implemented by every value that exposes a [meta.Bag] —
// [node.Node] and [emit.Node] both do.
type metaCarrier interface {
	Meta() *meta.Bag
}

// WithDirective returns a [Query] predicate matching values that
// carry the named directive. Use with [Query.Where]:
//
//	reader.Structs().Where(store.WithDirective[*node.Struct]("repo")).Each(...)
func WithDirective[T directiveCarrier](name directive.Name) func(T) bool {
	return func(v T) bool { return v.HasDirective(name) }
}

// WithMeta returns a [Query] predicate matching values whose
// metadata bag has the named key set (defaults are not consulted;
// only explicitly resolved values).
func WithMeta[T metaCarrier](name string) func(T) bool {
	return func(v T) bool { return v.Meta().Has(name) }
}

// WithMetaKey is the typed sibling of [WithMeta]: the predicate
// matches when the key resolves to a value (any non-tombstoned
// value at any authority).
func WithMetaKey[T metaCarrier, V any](k meta.Key[V]) func(T) bool {
	return func(v T) bool { return k.Has(v.Meta()) }
}

// MetaEq returns a predicate matching values where the typed key k
// resolves to a value equal to want. Tombstoned keys do not match.
func MetaEq[T metaCarrier, V comparable](k meta.Key[V], want V) func(T) bool {
	return func(v T) bool {
		got, ok := k.Get(v.Meta())
		return ok && got == want
	}
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
