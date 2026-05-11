// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

// Versioned is the optional plugin capability for declaring a plugin
// version string. The version composes into the plugin's cache key
// so a version bump invalidates that plugin's cached outputs without
// disturbing any other plugin's cache.
//
// Conventions:
//
//   - Stable plugins return a semantic-version string ("1.2.3").
//   - Plugins under active development may return a source-content
//     hash so cache invalidation tracks the actual implementation
//     bytes rather than a hand-maintained version field.
//   - The empty string is permitted and means "do not contribute to
//     the cache key" — equivalent to opting out of cache integration.
//
// The pipeline does not interpret the string; it is hashed into the
// cache key verbatim alongside the plugin's other inputs.
type Versioned interface {
	// Version returns the plugin's version identifier.
	Version() string
}
