// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cache_test

import "testing"

// assertNoError fails the test if err is non-nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
