// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Parser converts a raw string (typically the right-hand side of a
// +gen:meta directive) into a typed value. Parsers are attached to
// [Key] declarations so the directive-override step can stamp typed
// values without each plugin re-implementing parsing.
//
// A Parser returns the zero value and an error wrapping [ErrParse]
// when the input is malformed for T.
type Parser[T any] func(raw string) (T, error)

// ErrParse is the sentinel wrapped by built-in parsers when an input
// cannot be converted to the target type. Callers use [errors.Is]
// to detect parse failures and distinguish them from other errors.
var ErrParse = errors.New("meta: parse error")

// BoolParser parses "true" / "false" (case-insensitive) and the
// shorthand "1" / "0". Any other input returns [ErrParse].
//
// The shorthand "" is accepted as true so that a bare "+gen:foo bar"
// (no =value) stamps a bool key at true.
func BoolParser(raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "", "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("%w: bool: %q", ErrParse, raw)
	}
}

// StringParser returns the raw input unchanged. It exists so that
// every Key registration follows the same shape (Key[T] + Parser[T])
// regardless of whether parsing is needed.
func StringParser(raw string) (string, error) {
	return raw, nil
}

// IntParser parses a decimal int. Leading or trailing whitespace
// is rejected; callers that want lenient parsing supply their own.
func IntParser(raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: int: %q", ErrParse, raw)
	}
	return n, nil
}

// StringListParser parses a comma-separated list with no whitespace
// trimming around elements (callers wanting trimming should pre-
// process). An empty input yields an empty slice rather than nil so
// "the list was present and empty" round-trips cleanly through
// directive sources.
func StringListParser(raw string) ([]string, error) {
	if raw == "" {
		return []string{}, nil
	}
	return strings.Split(raw, ","), nil
}

// NodeRefParser is the spec-listed parser for keys whose value
// resolves to a node by name. The parser itself has no scope —
// it simply records the raw name as the value's textual form and
// returns [ErrParse] on the empty case. Callers that need to
// resolve the name to a concrete [node.Node] do so against the
// owning node's scope at the directive-override step (or later);
// the parser preserves the source-level identifier verbatim.
func NodeRefParser(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("%w: empty node ref", ErrParse)
	}
	return raw, nil
}
