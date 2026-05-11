// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

// assertNoError fails the test if err is non-nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// targetAt returns a populated [emit.Target] used by writer tests
// that need a routable destination.
func targetAt(dir, file string) emit.Target {
	return emit.Target{Dir: dir, Filename: file, Package: "x"}
}
