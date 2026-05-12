// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"go.thesmos.sh/eidos/core/position"
)

// writeErr writes a formatted error message to env.Stderr,
// terminated by a newline. Failures from the underlying writer
// are silently dropped: by the time the cli is writing to Stderr
// the process is already in an error path, and an error-on-error
// is a "give up cleanly" situation.
func writeErr(env *Env, format string, args ...any) {
	if env == nil || env.Stderr == nil {
		return
	}
	_, _ = fmt.Fprintf(env.Stderr, format+"\n", args...)
}

// encodeJSONLine writes v as a single JSON object plus a trailing
// newline. Used by the JSON-format renderers across commands.
// Writer errors propagate wrapped with "cli: encode json:" so
// callers can locate the failure context.
func encodeJSONLine(w io.Writer, v any) error {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("cli: encode json: %w", err)
	}
	return nil
}

// recoverInto is the panic-recovery defer body used by every
// command's Execute. Any non-nil panic converts to
// [ExitInternalError] with a structured diagnostic written to
// env.Stderr; the recovered exit code is stored back into the
// caller's *int via the named-return pattern.
func recoverInto(env *Env, exit *int) {
	r := recover()
	if r == nil {
		return
	}
	writeErr(env, "internal error: %v", r)
	if env != nil && env.Diag != nil {
		env.Diag.Internalf(position.Pos{}, "cli: recovered panic: %v", r)
	}
	*exit = ExitInternalError
}

// patternsFromConfig flattens cfg.Sources[*].Patterns into a
// single slice. Used by commands that drive the pipeline; the
// frontends receive every pattern.
func patternsFromConfig(cfg *Config) []string {
	if cfg == nil {
		return nil
	}
	var out []string
	for _, src := range cfg.Sources {
		out = append(out, src.Patterns...)
	}
	return out
}
