// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// JSONFormatVersion identifies the schema version of the JSON output
// produced by [JSONFormatter]. Consumers (CI runners, IDE plugins)
// branch on this when reading streamed diagnostics.
const JSONFormatVersion = 1

// JSONFormatter renders diagnostics as line-delimited JSON: one
// diagnostic per line, followed by a final aggregate summary object.
// The schema is stable and versioned via [JSONFormatVersion].
//
// Each diagnostic line marshals the [Diag] struct directly (Severity
// renders as a string; Pos as a nested object that is omitted when
// zero). The summary line has the shape:
//
//	{"version":1,"summary":{"error":N,"warn":N,"info":N,"internal":N}}
//
// The trailing newline after every line keeps the stream parseable
// by tools that ingest one JSON object per line.
type JSONFormatter struct{}

// jsonSummary is the trailing aggregate object emitted at the end of
// the stream. It does not include any per-diagnostic content; the
// individual lines that precede it carry the full record.
type jsonSummary struct {
	Version int           `json:"version"`
	Summary severityCount `json:"summary"`
}

// severityCount is the per-severity tally embedded in jsonSummary.
type severityCount struct {
	Error    int `json:"error"`
	Warn     int `json:"warn"`
	Info     int `json:"info"`
	Internal int `json:"internal"`
}

// Format writes one JSON object per diagnostic in insertion order,
// followed by a single summary object. Each object is terminated by
// '\n' so streamed consumers can read one line at a time.
//
// Output is composed into an internal buffer; the only IO error
// surfaces from the single final write to w.
func (JSONFormatter) Format(w io.Writer, diags []Diag) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, d := range diags {
		// bytes.Buffer writes never fail; the only Encode failures
		// come from un-marshalable values, which our schema rules out.
		_ = enc.Encode(d)
	}
	summary := jsonSummary{Version: JSONFormatVersion}
	for _, d := range diags {
		switch d.Severity {
		case Error:
			summary.Summary.Error++
		case Warn:
			summary.Summary.Warn++
		case Info:
			summary.Summary.Info++
		case Internal:
			summary.Summary.Internal++
		}
	}
	_ = enc.Encode(summary)
	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("diag: write JSON output: %w", err)
	}
	return nil
}
