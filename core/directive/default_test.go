// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
)

func TestDefaultPrefix(t *testing.T) {
	t.Parallel()

	t.Run("matches the project convention", func(t *testing.T) {
		t.Parallel()
		if directive.DefaultPrefix != "gen" {
			t.Fatalf("DefaultPrefix = %q, want %q", directive.DefaultPrefix, "gen")
		}
	})
}

func TestDefaultParser(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil parser configured for DefaultPrefix", func(t *testing.T) {
		t.Parallel()
		p := directive.DefaultParser()
		if p == nil {
			t.Fatalf("DefaultParser returned nil")
		}
		if got := p.Prefix(); got != directive.DefaultPrefix {
			t.Fatalf("DefaultParser prefix = %q, want %q", got, directive.DefaultPrefix)
		}
	})

	t.Run("returns the same singleton across calls", func(t *testing.T) {
		t.Parallel()
		first := directive.DefaultParser()
		second := directive.DefaultParser()
		if first != second {
			t.Fatalf("DefaultParser should return the same singleton")
		}
	})
}
