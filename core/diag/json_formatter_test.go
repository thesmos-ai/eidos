// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import (
	"encoding/json"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
)

func TestJSONFormatter_Format(t *testing.T) {
	t.Parallel()

	t.Run("emits one JSON object per diagnostic plus a summary", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Error, Plugin: "validation", Pos: position.At("auth.go", 18, 5), Message: "boom"},
			{Severity: diag.Warn, Plugin: "shape-writer", Pos: position.At("user.go", 42, 1), Message: "ish"},
			{Severity: diag.Info, Plugin: "p", Message: "consider"},
		}
		var b strings.Builder
		if err := (diag.JSONFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
		if got, want := len(lines), len(diags)+1; got != want {
			t.Fatalf("got %d lines, want %d (one per diagnostic plus summary)", got, want)
		}
		// Each per-diagnostic line round-trips to a Diag value.
		for i, raw := range lines[:len(diags)] {
			var d diag.Diag
			if err := json.Unmarshal([]byte(raw), &d); err != nil {
				t.Fatalf("line %d failed to round-trip: %v\n%s", i, err, raw)
			}
			if d != diags[i] {
				t.Fatalf("line %d round-tripped to %+v, want %+v", i, d, diags[i])
			}
		}
		// Summary line carries the version constant and per-severity counts.
		summary := struct {
			Version int `json:"version"`
			Summary struct {
				Error    int `json:"error"`
				Warn     int `json:"warn"`
				Info     int `json:"info"`
				Internal int `json:"internal"`
			} `json:"summary"`
		}{}
		if err := json.Unmarshal([]byte(lines[len(diags)]), &summary); err != nil {
			t.Fatalf("summary failed to unmarshal: %v\n%s", err, lines[len(diags)])
		}
		if summary.Version != diag.JSONFormatVersion {
			t.Fatalf("summary version = %d, want %d", summary.Version, diag.JSONFormatVersion)
		}
		switch {
		case summary.Summary.Error != 1,
			summary.Summary.Warn != 1,
			summary.Summary.Info != 1,
			summary.Summary.Internal != 0:
			t.Fatalf("summary counts wrong: %+v", summary.Summary)
		}
	})

	t.Run("empty input produces only the summary line", func(t *testing.T) {
		t.Parallel()
		var b strings.Builder
		if err := (diag.JSONFormatter{}).Format(&b, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
		if len(lines) != 1 {
			t.Fatalf("empty input should produce only the summary; got %d lines", len(lines))
		}
	})

	t.Run("Internal severity is counted", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{{Severity: diag.Internal, Message: "framework bug"}}
		var b strings.Builder
		if err := (diag.JSONFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(b.String(), `"internal":1`) {
			t.Fatalf("summary should count internal diagnostics; got\n%s", b.String())
		}
	})

	t.Run("propagates write errors", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{{Severity: diag.Error, Message: "boom"}}
		if err := (diag.JSONFormatter{}).Format(failingWriter{}, diags); err == nil {
			t.Fatalf("expected an error from failing writer")
		}
	})
}
