// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package naming_test

import (
	"slices"
	"testing"
)

// assertEqualSlices fails the test if got and want differ in length or
// any element. The diagnostic message identifies the first divergence.
func assertEqualSlices(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("slice mismatch:\n got:  %#v\n want: %#v", got, want)
	}
}

// assertEqualString is the string analogue, kept symmetrical with
// assertEqualSlices for readability.
func assertEqualString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string mismatch:\n got:  %q\n want: %q", got, want)
	}
}
