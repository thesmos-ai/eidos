// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cache

// Cache is the content-addressed key/value store the pipeline uses
// to memoise plugin outputs. Implementations are expected to be
// safe for concurrent use; concurrent Put calls with the same key
// are idempotent because the cache is content-addressed (the key
// determines the value).
//
// The interface is intentionally minimal — Get for lookups, Put for
// inserts. Higher-level concerns (retention, eviction, statistics)
// live outside the interface.
type Cache interface {
	// Get returns the cached bytes for key along with true; when
	// the key has no entry it returns nil and false. The returned
	// slice is safe for the caller to mutate.
	Get(key string) ([]byte, bool)

	// Put stores value under key. Returns a non-nil error only when
	// the underlying medium fails (filesystem error for [Disk];
	// [None] always returns nil).
	Put(key string, value []byte) error
}
