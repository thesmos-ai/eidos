// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package domain

// Numeric is the type-set union constraint exercised below. The
// union with `~` underlying-type entries verifies the
// renderUnion path: a generated codec or repository instantiated
// over Numeric should render the bracketed type-arg list and
// reference Numeric across packages.
type Numeric interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

// Comparable mirrors the stdlib `comparable` constraint as a named
// interface so test code can reference it cross-package without
// pulling in the unnamed constraint.
type Comparable interface {
	comparable
}

// Box is the canonical single-parameter generic envelope. The
// builder generator (when applied to a host containing a Box[T]
// field) must render the type argument intact (`Box[string]`,
// `Box[*User]`).
type Box[T any] struct {
	Value T
}

// Result is a constrained generic that pairs a value with an error.
// Its multi-method surface forces renderInterface / renderStruct to
// thread the type parameter through every method signature.
type Result[T any] struct {
	Value T
	Err   error
}

// IsOK reports whether the wrapped Err is nil. Method on a generic
// type exercises the receiver-type generic argument rendering.
func (r Result[T]) IsOK() bool { return r.Err == nil }

// Page is a two-parameter generic — the items and a cursor token —
// modelling pagination over any entity. Used by storage.Repository's
// List method so the rendered repository carries `Page[T, string]`
// or similar instantiations in its return slots.
type Page[T any, Cursor any] struct {
	Items  []T
	Cursor Cursor
}

// Filter is a generic constructor for a typed predicate function.
// Anonymous-type-parameter forms appear in API handlers below.
type Filter[T any] func(T) bool

// Map is a multi-parameter generic with a `comparable` constraint —
// the same shape stdlib `map[K]V` operates on but expressed as a
// named generic so consumers can reference it qualified.
type Map[K comparable, V any] struct {
	Entries map[K]V
}

// Get exercises constrained-parameter method rendering: the receiver
// uses K's `comparable` constraint and V's `any` constraint.
func (m Map[K, V]) Get(k K) (V, bool) {
	v, ok := m.Entries[k]
	return v, ok
}

// Sum is a top-level generic function constrained on Numeric. The
// frontend produces it as a function-level type-parameter list; the
// renderer must propagate `[T Numeric]` to the rendered signature
// when buildergen or any future generator emits forwarding wrappers.
func Sum[T Numeric](values []T) T {
	var zero T
	for _, v := range values {
		zero += v
	}
	return zero
}
