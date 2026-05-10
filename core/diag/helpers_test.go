// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import "testing"

// assertEqualString fails the test if got and want differ.
func assertEqualString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string mismatch:\n got:  %q\n want: %q", got, want)
	}
}
