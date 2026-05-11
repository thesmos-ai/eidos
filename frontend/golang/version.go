// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import "go.thesmos.sh/eidos/emit"

// FrontendVersion is the Go frontend's own version identifier. The
// pipeline composes the version into the frontend's cache key so
// bumping this constant invalidates every cached package the
// frontend previously produced — appropriate when a conversion bug
// or shape change in the frontend would make older cache entries
// incorrect.
//
// The version is independent from [emit.Version] (which tracks the
// emit-graph contract) — the frontend produces nodes, not emit, so
// only its own conversion code affects this value.
const FrontendVersion = "1.0.0"

// supportedEmitVersions lists the emit major versions the frontend
// is compatible with. The frontend itself does not produce emit
// values, but it participates in the pipeline's emit-version
// compatibility check via the [plugin.EmitVersioned] capability so
// users get a positioned diagnostic at Build time if they pair the
// frontend with an emit major it does not understand.
var supportedEmitVersions = []string{emit.Major()}
