// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package doc_test exists so the package's doc-only file participates
// in the project standard of one source file → one test file. The
// import below keeps the dependency edge live and prevents the file
// from being mistaken for unused.
package golang_test

import (
	"testing"

	"go.thesmos.sh/eidos/frontend/golang"
)

// TestPackageDoc is the doc.go pair: the file declares no symbols of
// its own beyond the package documentation; the pair asserts the
// package's identifier surface stays reachable from a _test importer.
func TestPackageDoc(t *testing.T) {
	t.Parallel()
	t.Run("package symbols are reachable", func(t *testing.T) {
		t.Parallel()
		if golang.FrontendName == "" {
			t.Fatalf("FrontendName must not be empty")
		}
		if golang.FrontendVersion == "" {
			t.Fatalf("FrontendVersion must not be empty")
		}
	})
}
