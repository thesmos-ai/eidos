// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
)

// failingWriter is an io.Writer whose every Write returns errFailWrite.
// Used to exercise the formatter's error paths.
type failingWriter struct{}

var errFailWrite = errors.New("test: write failed")

func (failingWriter) Write([]byte) (int, error) { return 0, errFailWrite }

func TestTextFormatter_Format(t *testing.T) {
	t.Parallel()

	t.Run("renders aligned rows with summary by default", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{
				Severity: diag.Warn,
				Plugin:   "shape-writer",
				Pos:      position.At("user.go", 42, 1),
				Message:  "detected Writer",
			},
			{
				Severity: diag.Error,
				Plugin:   "validation",
				Pos:      position.At("auth.go", 18, 5),
				Message:  "field secret cannot be omitempty",
			},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		for _, want := range []string{
			"warn", "shape-writer", "user.go:42:1", "detected Writer",
			"error", "validation", "auth.go:18:5", "field secret cannot be omitempty",
			"1 error", "1 warning", "run completed; output not written due to errors",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("output missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("hides Info by default", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Info, Plugin: "p", Message: "considered"},
			{Severity: diag.Warn, Plugin: "p", Message: "watch out"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		if strings.Contains(out, "considered") {
			t.Fatalf("Info line should be hidden by default; got:\n%s", out)
		}
		if !strings.Contains(out, "info hidden") {
			t.Fatalf("summary should note hidden Info diagnostics; got:\n%s", out)
		}
	})

	t.Run("verbose surfaces Info diagnostics", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Info, Plugin: "p", Message: "considered"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{Verbose: true}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(b.String(), "considered") {
			t.Fatalf("Info line should appear under Verbose; got:\n%s", b.String())
		}
	})

	t.Run("emits dash placeholder for missing plugin or pos", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Warn, Message: "global warning"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, "-") {
			t.Fatalf("expected dash placeholder in output:\n%s", out)
		}
	})

	t.Run("Internal severity contributes to the not-written suffix", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Internal, Message: "framework bug"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, "1 internal error") {
			t.Fatalf("expected Internal count in summary:\n%s", out)
		}
		if !strings.Contains(out, "run completed; output not written") {
			t.Fatalf("Internal severity should trigger the not-written suffix:\n%s", out)
		}
	})

	t.Run("Detail is indented and follows the row", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Error, Plugin: "p", Message: "boom", Detail: "stack line one\nstack line two"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, "    stack line one") {
			t.Fatalf("detail line should be indented; got:\n%s", out)
		}
	})

	t.Run("Detail without trailing newline still terminates the line", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Error, Message: "boom", Detail: "single line"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, "    single line\n") {
			t.Fatalf("detail without trailing newline should still terminate:\n%q", out)
		}
	})

	t.Run("empty diag slice produces no summary", func(t *testing.T) {
		t.Parallel()
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Len() != 0 {
			t.Fatalf("empty input should produce no output; got %q", b.String())
		}
	})

	t.Run("HideSummary suppresses the trailing summary", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Error, Message: "boom"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{HideSummary: true}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		if strings.Contains(out, "1 error") || strings.Contains(out, "run completed") {
			t.Fatalf("HideSummary should suppress the summary line; got:\n%s", out)
		}
	})

	t.Run("warnings without errors do not append the not-written suffix", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Warn, Message: "just a warning"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		if strings.Contains(out, "not written") {
			t.Fatalf("warning-only output should not include the not-written suffix:\n%s", out)
		}
	})

	t.Run("multiple errors render with plural noun", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{
			{Severity: diag.Error, Message: "first"},
			{Severity: diag.Error, Message: "second"},
			{Severity: diag.Warn, Message: "third"},
			{Severity: diag.Warn, Message: "fourth"},
		}
		var b strings.Builder
		if err := (diag.TextFormatter{}).Format(&b, diags); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := b.String()
		if !strings.Contains(out, "2 errors") {
			t.Fatalf("plural error count missing from summary:\n%s", out)
		}
		if !strings.Contains(out, "2 warnings") {
			t.Fatalf("plural warning count missing from summary:\n%s", out)
		}
	})

	t.Run("propagates write errors", func(t *testing.T) {
		t.Parallel()
		diags := []diag.Diag{{Severity: diag.Error, Message: "boom"}}
		if err := (diag.TextFormatter{}).Format(failingWriter{}, diags); err == nil {
			t.Fatalf("expected an error from failing writer")
		}
	})
}
