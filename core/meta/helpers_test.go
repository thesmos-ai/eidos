// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta_test

import (
	"testing"
)

// assertEqualString fails the test if got and want differ.
func assertEqualString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string mismatch:\n got:  %q\n want: %q", got, want)
	}
}

// assertNoError fails the test if err is non-nil. The message
// includes context for debugging without forcing every call site to
// write a t.Fatalf.
func assertNoError(t *testing.T, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", context, err)
	}
}
