// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"go/format"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// formatBody runs body through [go/format.Source] and returns the
// canonical Go-formatted result. Format failure is non-fatal: the
// function attaches a [diag.Severity] Warn diagnostic to ps naming
// target and the underlying error, and returns body unchanged so
// the unformatted output still reaches the sink for downstream
// inspection.
//
// The diagnostic carries [position.Pos]{} — format errors reference
// positions in the rendered text, which don't round-trip cleanly to
// source positions, so attaching at the Target level is the
// authoritative attribution.
func formatBody(body []byte, target emit.Target, ps *diag.PluginSink) []byte {
	formatted, err := format.Source(body)
	if err != nil {
		ps.Warnf(position.Pos{}, "%s: format.Source failed: %v", target.JoinPath(), err)
		return body
	}
	return formatted
}
