// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package cache is the content-addressed store the pipeline uses to
// memoise plugin outputs between runs. A cache key is the hash of
// every input that determined the cached value — the plugin's
// version plus the [store.ReadSet] it observed — so a fresh hit is
// equivalent to a fresh recompute.
//
// The package exposes the [Cache] interface and two implementations:
//
//   - [None] is a no-op cache (every Get is a miss; Put is a no-op).
//     Use for hermetic CI runs or to disable caching entirely.
//   - [Disk] is a filesystem-backed cache rooted at a configurable
//     directory. Entries are immutable and written atomically via
//     temp+rename so concurrent writers cannot observe a partial
//     file.
//
// Entries never change in place — different inputs produce different
// keys, so the cache grows monotonically. Periodic eviction by
// retention policy lives at a higher layer (the pipeline run loop).
package cache
