// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest_test

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

// targetAt returns a populated [emit.Target] used by manifest tests.
func targetAt(dir, file string) emit.Target {
	return emit.Target{Dir: dir, Filename: file, Package: "x"}
}

// targetAtPath returns a populated [emit.Target] with an
// explicit ImportPath — required by tests that exercise the
// scope-aware Prune filter (which keys on Target.ImportPath).
func targetAtPath(dir, file, importPath string) emit.Target {
	return emit.Target{Dir: dir, Filename: file, Package: "x", ImportPath: importPath}
}
