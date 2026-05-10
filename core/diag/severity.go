// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

import (
	"errors"
	"fmt"
	"strconv"
)

// Severity ranks diagnostics from least to most severe.
//
// Info / Warn / Error are the user-facing levels; Internal is reserved
// for core or framework bugs (panics in core packages, contract
// violations) so they surface with a distinct prefix and contribute to
// a separate CLI exit code.
type Severity int

// The severity levels in ascending order.
const (
	// Info is informational output, hidden by default and shown via
	// --verbose. Use for "considered but skipped" notes and for
	// summaries that aid debugging without indicating a problem.
	Info Severity = iota
	// Warn flags something probably wrong that does not block the
	// pipeline. Output is still produced; the run exits 0.
	Warn
	// Error blocks output: when any Error diagnostic is emitted, the
	// pipeline runs every remaining plugin to completion and then
	// returns a non-zero exit. The user's code is wrong.
	Error
	// Internal reports a bug in core or a plugin's framework
	// contract — a panic in a core package, a violated invariant, an
	// impossible state. CLI exit code distinguishes Internal from
	// user-side Error so operators can route the report correctly.
	Internal
)

// ErrUnknownSeverity is returned by [ParseSeverity] for an input that
// does not name a defined level.
var ErrUnknownSeverity = errors.New("diag: unknown severity")

// String returns the lower-case textual form of s. Unknown values
// stringify as "severity(N)" so logs make the bad value visible
// without panicking.
func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Error:
		return "error"
	case Internal:
		return "internal"
	default:
		return "severity(" + strconv.Itoa(int(s)) + ")"
	}
}

// ParseSeverity returns the Severity matching the lower-case textual
// form (the same form [Severity.String] produces for known values).
// Unknown inputs return [ErrUnknownSeverity] wrapped with the offending
// string.
func ParseSeverity(s string) (Severity, error) {
	switch s {
	case "info":
		return Info, nil
	case "warn":
		return Warn, nil
	case "error":
		return Error, nil
	case "internal":
		return Internal, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrUnknownSeverity, s)
	}
}

// MarshalJSON renders s as a JSON string using its textual form, so
// downstream consumers see "error" rather than the integer value.
func (s Severity) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, s.String()), nil
}

// UnmarshalJSON parses the JSON string form produced by MarshalJSON.
// It returns [ErrUnknownSeverity] for any other input.
func (s *Severity) UnmarshalJSON(data []byte) error {
	str, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("diag: severity must be a JSON string: %w", err)
	}
	parsed, err := ParseSeverity(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
