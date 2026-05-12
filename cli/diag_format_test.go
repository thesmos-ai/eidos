// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
)

// TestRenderDiagnostics_Text covers the text renderer: one line
// per diagnostic, position included when non-zero, plugin tag in
// brackets, detail block indented when present.
func TestRenderDiagnostics_Text(t *testing.T) {
	t.Parallel()

	t.Run("error with position and plugin renders the canonical text form", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.For("repogen").Errorf(position.At("user.go", 42, 8), "duplicate field %q", "Email")
		var buf bytes.Buffer
		if err := cli.RenderDiagnostics(&buf, s, cli.DiagFormatText, false, false); err != nil {
			t.Fatalf("RenderDiagnostics: %v", err)
		}
		want := `error user.go:42:8 [repogen]: duplicate field "Email"` + "\n"
		if buf.String() != want {
			t.Fatalf("text output mismatch:\n  got: %q\n want: %q", buf.String(), want)
		}
	})

	t.Run("zero position omits the position field", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.For("pipeline").Warnf(position.Pos{}, "manifest write skipped")
		var buf bytes.Buffer
		if err := cli.RenderDiagnostics(&buf, s, cli.DiagFormatText, false, false); err != nil {
			t.Fatalf("RenderDiagnostics: %v", err)
		}
		want := "warn [pipeline]: manifest write skipped\n"
		if buf.String() != want {
			t.Fatalf("text output mismatch:\n  got: %q\n want: %q", buf.String(), want)
		}
	})

	t.Run("info is suppressed by default and shown with verbose", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Infof(position.Pos{}, "started")
		var buf bytes.Buffer
		if err := cli.RenderDiagnostics(&buf, s, cli.DiagFormatText, false, false); err != nil {
			t.Fatalf("RenderDiagnostics: %v", err)
		}
		if buf.Len() != 0 {
			t.Fatalf("info should be suppressed without verbose; got %q", buf.String())
		}
		buf.Reset()
		if err := cli.RenderDiagnostics(&buf, s, cli.DiagFormatText, true, false); err != nil {
			t.Fatalf("RenderDiagnostics(verbose): %v", err)
		}
		if !strings.Contains(buf.String(), "info") {
			t.Fatalf("info should render with verbose; got %q", buf.String())
		}
	})

	t.Run("quiet suppresses warn but not error", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Warnf(position.Pos{}, "noisy")
		s.Errorf(position.Pos{}, "fatal")
		var buf bytes.Buffer
		if err := cli.RenderDiagnostics(&buf, s, cli.DiagFormatText, false, true); err != nil {
			t.Fatalf("RenderDiagnostics(quiet): %v", err)
		}
		got := buf.String()
		if strings.Contains(got, "noisy") {
			t.Fatalf("warn should be suppressed under quiet; got %q", got)
		}
		if !strings.Contains(got, "fatal") {
			t.Fatalf("error should always render; got %q", got)
		}
	})

	t.Run("detail block renders indented under the line", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.For("debug").AppendDetail(diag.Error, position.Pos{}, "boom", "stack-frame-1\nstack-frame-2")
		var buf bytes.Buffer
		if err := cli.RenderDiagnostics(&buf, s, cli.DiagFormatText, false, false); err != nil {
			t.Fatalf("RenderDiagnostics: %v", err)
		}
		got := buf.String()
		if !strings.Contains(got, "    stack-frame-1") {
			t.Fatalf("detail should be indented; got %q", got)
		}
	})

	t.Run("nil sink is a no-op", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := cli.RenderDiagnostics(&buf, nil, cli.DiagFormatText, false, false); err != nil {
			t.Fatalf("RenderDiagnostics(nil) = %v", err)
		}
		if buf.Len() != 0 {
			t.Fatalf("nil sink should write nothing; got %q", buf.String())
		}
	})
}

// TestRenderDiagnostics_JSON covers the NDJSON renderer: one JSON
// object per diagnostic, each line independently parseable, schema
// preserves [diag.Diag] field tags.
func TestRenderDiagnostics_JSON(t *testing.T) {
	t.Parallel()

	t.Run("two diagnostics produce two NDJSON lines", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.For("repogen").Errorf(position.At("user.go", 42, 8), "duplicate")
		s.For("repogen").Warnf(position.Pos{}, "deprecated option")
		var buf bytes.Buffer
		if err := cli.RenderDiagnostics(&buf, s, cli.DiagFormatJSON, false, false); err != nil {
			t.Fatalf("RenderDiagnostics: %v", err)
		}
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 NDJSON lines; got %d (%q)", len(lines), buf.String())
		}
		var first diag.Diag
		if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
			t.Fatalf("first line not valid JSON: %v", err)
		}
		if first.Severity != diag.Error || first.Plugin != "repogen" || first.Pos.File != "user.go" {
			t.Fatalf("first line shape unexpected: %+v", first)
		}
	})

	t.Run("quiet still suppresses warn in JSON mode", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Warnf(position.Pos{}, "skip")
		s.Errorf(position.Pos{}, "keep")
		var buf bytes.Buffer
		if err := cli.RenderDiagnostics(&buf, s, cli.DiagFormatJSON, false, true); err != nil {
			t.Fatalf("RenderDiagnostics: %v", err)
		}
		if strings.Contains(buf.String(), "skip") {
			t.Fatalf("warn should be suppressed; got %q", buf.String())
		}
		if !strings.Contains(buf.String(), "keep") {
			t.Fatalf("error should render; got %q", buf.String())
		}
	})
}
