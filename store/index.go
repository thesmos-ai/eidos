// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"fmt"
	"sync"
)

// Bucket is an insertion-ordered, qualified-name-keyed collection of
// T. Each per-kind store ([NodeView.Structs], [NodeView.Methods], …)
// uses a Bucket internally: a slice of items captures the insertion
// order for deterministic iteration, and a parallel map provides
// O(1) qualified-name lookup. Concurrent-safe via RWMutex.
//
// Bucket is exported so the [NodeView] / [EmitView] accessors can
// surface the same shape per kind without repeating the storage code
// in every per-kind file.
type Bucket[T any] struct {
	mu      sync.RWMutex
	items   []T
	byQName map[string]T
}

// NewBucket returns an empty Bucket ready for use.
func NewBucket[T any]() *Bucket[T] {
	return &Bucket[T]{byQName: map[string]T{}}
}

// Add records item under qname. Returns [ErrDuplicateQName] when
// qname is already present.
func (b *Bucket[T]) Add(qname string, item T) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, dup := b.byQName[qname]; dup {
		return fmt.Errorf("%w: %q", ErrDuplicateQName, qname)
	}
	b.items = append(b.items, item)
	b.byQName[qname] = item
	return nil
}

// ByQName returns the item recorded under qname, or the zero value
// and false when no item matches.
func (b *Bucket[T]) ByQName(qname string) (T, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	v, ok := b.byQName[qname]
	return v, ok
}

// Items returns a copy of the bucket's items in insertion order. The
// returned slice is safe for the caller to mutate and iterate while
// the bucket continues to accept writes.
func (b *Bucket[T]) Items() []T {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]T, len(b.items))
	copy(out, b.items)
	return out
}

// Len returns the number of items in the bucket.
func (b *Bucket[T]) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.items)
}

// Range invokes fn for each item in insertion order. Returning false
// from fn stops iteration. The iteration holds the read lock for its
// duration; fn must not call back into write methods on the same
// bucket.
func (b *Bucket[T]) Range(fn func(T) bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, item := range b.items {
		if !fn(item) {
			return
		}
	}
}

// MultiIndex is a generic many-to-one keyed index: many keys, each
// mapping to an insertion-ordered list of values. Used for
// cross-cutting indices like "by directive presence" and "by
// declaring package" where one key has many entries. Concurrent-safe
// via RWMutex.
type MultiIndex[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K][]V
}

// NewMultiIndex returns an empty MultiIndex ready for use.
func NewMultiIndex[K comparable, V any]() *MultiIndex[K, V] {
	return &MultiIndex[K, V]{entries: map[K][]V{}}
}

// Add appends value to the list under key, preserving insertion
// order.
func (m *MultiIndex[K, V]) Add(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[key] = append(m.entries[key], value)
}

// Get returns a copy of the values recorded under key in insertion
// order. Returns nil when the key has no entries.
func (m *MultiIndex[K, V]) Get(key K) []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src, ok := m.entries[key]
	if !ok {
		return nil
	}
	out := make([]V, len(src))
	copy(out, src)
	return out
}

// Has reports whether key has at least one recorded value.
func (m *MultiIndex[K, V]) Has(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.entries[key]
	return ok
}

// Len returns the number of distinct keys in the index.
func (m *MultiIndex[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}
