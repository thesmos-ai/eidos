// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"go.thesmos.sh/eidos/core/diag"
)

// RenderDiagnostics writes every entry recorded by s to w in the
// requested format. verbose includes Info diagnostics; quiet
// suppresses Warn. Errors always render regardless of either flag.
//
// Text format: one `<sev> <pos> [<plugin>] <message>` line per
// entry, with the Detail (when present) appended as an indented
// block.
//
// JSON format: one NDJSON object per entry matching the
// [diag.Diag] field shape — `severity`, `plugin`, `pos`,
// `message`, `detail`. Empty fields are omitted.
//
// Errors from the underlying writer are returned to the caller;
// the function does not buffer.
func RenderDiagnostics(w io.Writer, s *diag.Sink, format DiagFormat, verbose, quiet bool) error {
	if s == nil {
		return nil
	}
	for _, d := range s.Diagnostics() {
		if quiet && d.Severity == diag.Warn {
			continue
		}
		if !verbose && d.Severity == diag.Info {
			continue
		}
		var err error
		switch format {
		case DiagFormatJSON:
			err = renderDiagJSON(w, d)
		default:
			err = renderDiagText(w, d)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// renderDiagText formats one [diag.Diag] as a text line plus an
// optional indented detail block. [position.Pos.String] supplies
// the canonical `file:line:col` rendering — the formatter does
// no position parsing of its own. Writer errors propagate wrapped
// with the diagnostic-render context so the caller can locate the
// failure relative to the rendering loop.
func renderDiagText(w io.Writer, d diag.Diag) error {
	sev := severityLabel(d.Severity)
	pos := d.Pos.String()
	plugin := ""
	if d.Plugin != "" {
		plugin = " [" + d.Plugin + "]"
	}
	var err error
	if pos == "" {
		_, err = fmt.Fprintf(w, "%s%s: %s\n", sev, plugin, d.Message)
	} else {
		_, err = fmt.Fprintf(w, "%s %s%s: %s\n", sev, pos, plugin, d.Message)
	}
	if err != nil {
		return fmt.Errorf("cli: render text diagnostic: %w", err)
	}
	if d.Detail != "" {
		if _, err := fmt.Fprintf(w, "    %s\n", d.Detail); err != nil {
			return fmt.Errorf("cli: render diagnostic detail: %w", err)
		}
	}
	return nil
}

// renderDiagJSON writes one NDJSON object per entry. The schema
// matches the [diag.Diag] field tags so the on-disk shape stays
// stable across releases.
func renderDiagJSON(w io.Writer, d diag.Diag) error {
	if err := json.NewEncoder(w).Encode(d); err != nil {
		return fmt.Errorf("cli: render json diagnostic: %w", err)
	}
	return nil
}

// severityLabel returns the text label for a severity used in the
// text formatter.
func severityLabel(s diag.Severity) string {
	switch s {
	case diag.Error:
		return "error"
	case diag.Warn:
		return "warn"
	case diag.Info:
		return "info"
	default:
		return "diag"
	}
}
