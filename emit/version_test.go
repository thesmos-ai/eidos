// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("Version is a non-empty semver-shaped string", func(t *testing.T) {
		t.Parallel()
		if emit.Version == "" {
			t.Fatalf("Version should be non-empty")
		}
		if !strings.Contains(emit.Version, ".") {
			t.Fatalf("Version should contain '.' as a semver separator; got %q", emit.Version)
		}
	})
}

func TestMajor(t *testing.T) {
	t.Parallel()

	t.Run("returns the leading numeric segment of Version", func(t *testing.T) {
		t.Parallel()
		got := emit.Major()
		head, _, _ := strings.Cut(emit.Version, ".")
		if got != head {
			t.Fatalf("Major = %q, want %q", got, head)
		}
	})
}
