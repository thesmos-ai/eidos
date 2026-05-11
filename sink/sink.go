// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink

import "go.thesmos.sh/eidos/emit"

// Sink is the contract a backend uses to write rendered output. The
// backend groups [emit] entities by their [emit.Target] and calls
// [Sink.Write] once per target with the finalised file content
// (post-formatting, post-import-resolution).
//
// Implementations are responsible for atomicity (a partial write
// must not leave the destination in an inconsistent state),
// concurrent-safe Write (the backend renders files in parallel and
// dispatches into a shared sink), and deterministic flush ordering
// when ordering is observable.
//
// Errors from Write surface to the backend, which converts them to
// positioned [diag.Error] diagnostics and continues with the next
// target.
type Sink interface {
	// Write commits content under the destination identified by
	// target. Calls are serialised per target by the backend; an
	// implementation that buffers writes must flush before any
	// subsequent read of the same target.
	Write(target emit.Target, content []byte) error
}
