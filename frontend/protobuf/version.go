// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf

// FrontendVersion is the protobuf frontend's own version identifier.
// The string composes into the frontend's per-plugin cache key
// alongside the resolved descriptor set's content hash; bumping the
// constant invalidates every cache entry the frontend produced
// under the previous version, picking up changes to the converter's
// translation rules (new meta keys, changed type-mapping conventions,
// etc.) without disturbing other plugins' caches.
//
// The version is a semantic-version string. Bumps follow the
// emit-contract evolution rules documented on [plugin.Versioned]:
// patch for bug fixes that preserve every documented meta key,
// minor for additive meta-key additions, major for breaking changes
// to the documented vocabulary.
const FrontendVersion = "1.0.0"
