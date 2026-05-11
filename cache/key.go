// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// NewKey composes a canonical cache key from the supplied parts.
// Parts are joined by a colon — the conventional separator that
// keeps the human-readable prefix structure ("plugin:foo:v1:abc...")
// while remaining unambiguous in cache-backend storage. Empty parts
// are dropped so callers can pass conditionally-empty qualifiers
// without producing keys with collapsing or doubled separators.
//
// Typical shape: NewKey("plugin", pluginName, "version", pluginVersion, "input", inputHash).
// The format is convention, not policy — every cache implementation
// treats the result as an opaque string.
func NewKey(parts ...string) string {
	nonEmpty := parts[:0]
	for _, p := range parts {
		if p == "" {
			continue
		}
		nonEmpty = append(nonEmpty, p)
	}
	return strings.Join(nonEmpty, ":")
}

// HashBytes returns the lower-case hex SHA-256 digest of b. The
// digest is the conventional "input hash" component in a cache key
// — frontends feed it the concatenated file bytes; annotators and
// generators feed it their per-plugin read-set hashes.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// HashStrings returns the SHA-256 hex digest of the sorted, NUL-
// joined input. Used to compose order-insensitive hashes from a
// set of identifiers — e.g. the file paths contributing to a
// frontend's per-package input set. Sorting before hashing keeps
// the digest deterministic regardless of caller-supplied order.
func HashStrings(items []string) string {
	sorted := append([]string(nil), items...)
	sort.Strings(sorted)
	return HashBytes([]byte(strings.Join(sorted, "\x00")))
}
