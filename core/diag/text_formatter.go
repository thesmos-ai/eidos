// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// TextFormatter renders diagnostics for a human reader on a terminal.
//
// Output layout (one diagnostic per line, columns aligned):
//
//	severity   plugin           pos             message
//
// followed by a final summary line counting Error / Warn / Info /
// Internal totals and stating whether the run produced output.
//
// Info diagnostics are suppressed by default; setting Verbose=true
// surfaces them. Internal diagnostics are always shown regardless of
// Verbose because they indicate a framework bug.
//
// The zero value is usable and renders concisely; configure Verbose
// before calling Format.
type TextFormatter struct {
	// Verbose, when true, includes Info diagnostics in the output.
	Verbose bool
	// HideSummary, when true, skips the trailing summary line.
	// Useful when the formatter's output is being merged with other
	// content that supplies its own status footer.
	HideSummary bool
}

// Format writes diags to w in human-readable form. The result is
// deterministic: same diags slice produces byte-identical output.
//
// Output is composed into an internal buffer first, so the only IO
// error surfaces from the single final write to w.
func (f TextFormatter) Format(w io.Writer, diags []Diag) error {
	var buf bytes.Buffer
	visible := f.filter(diags)
	if len(visible) > 0 {
		tw := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
		for _, d := range visible {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				d.Severity, plainOrDash(d.Plugin), plainOrDash(d.Pos.String()), d.Message)
			if d.Detail != "" {
				writeDetail(tw, d.Detail)
			}
		}
		// bytes.Buffer writes never fail; tabwriter.Flush propagates
		// only buffer errors, so this is always nil.
		_ = tw.Flush()
	}
	if !f.HideSummary {
		f.writeSummary(&buf, diags)
	}
	if buf.Len() == 0 {
		return nil
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("diag: write text output: %w", err)
	}
	return nil
}

// filter drops Info diagnostics when Verbose is false. Internal and
// Warn / Error always pass through.
func (f TextFormatter) filter(diags []Diag) []Diag {
	if f.Verbose {
		return diags
	}
	out := make([]Diag, 0, len(diags))
	for _, d := range diags {
		if d.Severity == Info {
			continue
		}
		out = append(out, d)
	}
	return out
}

// writeSummary appends the trailing line listing counts of each
// severity. Counts are computed over the unfiltered input so the user
// sees the true picture even when running without --verbose. Writes
// to a bytes.Buffer never fail.
func (f TextFormatter) writeSummary(buf *bytes.Buffer, diags []Diag) {
	var errs, warns, infos, internals int
	for _, d := range diags {
		switch d.Severity {
		case Error:
			errs++
		case Warn:
			warns++
		case Info:
			infos++
		case Internal:
			internals++
		}
	}
	parts := make([]string, 0, 4)
	if errs > 0 {
		parts = append(parts, plural(errs, "error"))
	}
	if warns > 0 {
		parts = append(parts, plural(warns, "warning"))
	}
	if internals > 0 {
		parts = append(parts, plural(internals, "internal error"))
	}
	if !f.Verbose && infos > 0 {
		parts = append(parts, plural(infos, "info hidden")+" (run with --verbose to show)")
	}
	if len(parts) == 0 {
		return
	}
	suffix := ""
	if errs > 0 || internals > 0 {
		suffix = " (run completed; output not written due to errors)"
	}
	fmt.Fprintf(buf, "%s%s\n", strings.Join(parts, ", "), suffix)
}

// plainOrDash returns "-" when s is empty, otherwise s. Keeps the
// tabwriter columns aligned even when Plugin or Pos is missing.
func plainOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// plural returns the canonical "N noun(s)" form. Used by the summary
// line to avoid noisy "1 errors" output.
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// writeDetail indents detail content under the diagnostic line so the
// trailing tabwriter alignment is not disrupted. Writes to a
// bytes.Buffer never fail.
func writeDetail(buf io.Writer, detail string) {
	for line := range strings.Lines(detail) {
		fmt.Fprintf(buf, "    %s", line)
	}
	if !strings.HasSuffix(detail, "\n") {
		_, _ = io.WriteString(buf, "\n")
	}
}
