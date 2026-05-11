// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"testing"

	"go.thesmos.sh/eidos/frontend/golang"
)

// TestFrontendVersion covers the version constant the frontend
// exposes for cache-key composition.
func TestFrontendVersion(t *testing.T) {
	t.Parallel()

	t.Run("is non-empty", func(t *testing.T) {
		t.Parallel()
		if golang.FrontendVersion == "" {
			t.Fatalf("FrontendVersion must not be empty")
		}
	})
}
