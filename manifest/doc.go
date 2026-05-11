// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package manifest is the per-run record of every output file the
// pipeline produced, attributed to the plugins that contributed and
// hashed for change detection. Each pipeline run writes a manifest
// to ".eidos/manifest.json" under the workdir; subsequent runs read
// the previous manifest to compute prune candidates — files that the
// previous run produced but the current configuration no longer
// claims.
//
// The package exposes [Manifest] and [Output] data types, [Read] /
// [Write] for JSON I/O (atomic via temp+rename), and [Prune] for
// the diff that drives "eidos prune" and the CI drift check.
//
// Manifests are versioned at the document root; older readers
// detect future versions and refuse to interpret them rather than
// silently mis-parsing.
package manifest
