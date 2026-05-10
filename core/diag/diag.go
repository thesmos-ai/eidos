// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

import "go.thesmos.sh/eidos/core/position"

// Diag is a single positioned diagnostic produced by a plugin or by
// the pipeline runtime.
//
// The fields are stable for JSON serialisation: Severity is rendered
// as its textual form ("info", "warn", "error", "internal"); Pos is
// omitted from output when zero (plugin-global diagnostics); Plugin
// is omitted when empty (runtime-emitted diagnostics with no owning
// plugin). Message is the primary human-readable summary; Detail
// carries optional multi-line context (stack traces, expected vs
// actual values).
type Diag struct {
	Severity Severity     `json:"severity"`
	Plugin   string       `json:"plugin,omitempty"`
	Pos      position.Pos `json:"pos,omitzero"`
	Message  string       `json:"message"`
	Detail   string       `json:"detail,omitempty"`
}
