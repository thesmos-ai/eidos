// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package priority defines the [Priority] integer used by plugins
// that opt into capability ordering. Plugins implementing
// [plugin.CapabilityProvider] return a Priority; the pipeline groups
// them into priority buckets and orders buckets strictly by Priority
// value. Within a bucket, plugins are ordered by capability topo
// sort over Provides / Requires.
//
// Predefined buckets cover the annotator and generator phase
// conventions used by the reference plugins. Plugin authors can also
// pick custom values; the pipeline treats the integer literally
// without semantic interpretation beyond ordering.
package priority
