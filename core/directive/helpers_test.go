// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
)

// parseFirst is the convenience helper used by tests that expect a
// single-directive parse: it calls [directive.Parser.Parse],
// asserts a non-empty slice, and returns the first entry. Tests
// exercising multi-directive lines call [directive.Parser.Parse]
// directly and inspect the slice in full.
func parseFirst(t *testing.T, p *directive.Parser, body string, pos position.Pos) *directive.Directive {
	t.Helper()
	ds, err := p.Parse(body, pos)
	assertNoError(t, err, "Parse")
	if len(ds) == 0 {
		t.Fatalf("Parse(%q) returned no directives", body)
	}
	return ds[0]
}

// parseFirstComment is parseFirst's [directive.Parser.ParseComment]
// equivalent — calls the comment-stripping entry point and returns
// the first directive, failing the test when ParseComment returns
// nil for a directive comment that should have parsed.
func parseFirstComment(t *testing.T, p *directive.Parser, comment string, pos position.Pos) *directive.Directive {
	t.Helper()
	ds, err := p.ParseComment(comment, pos)
	assertNoError(t, err, "ParseComment")
	if len(ds) == 0 {
		t.Fatalf("ParseComment(%q) returned no directives", comment)
	}
	return ds[0]
}

// assertNoError fails the test with context when err is non-nil.
func assertNoError(t *testing.T, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", context, err)
	}
}

// assertEqualString fails the test if got and want differ.
func assertEqualString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string mismatch:\n got:  %q\n want: %q", got, want)
	}
}
