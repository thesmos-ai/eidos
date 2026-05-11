// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestTag_Kind(t *testing.T) {
	t.Parallel()

	t.Run("Tag reports KindTag", func(t *testing.T) {
		t.Parallel()
		var tg emit.Tag
		if got := tg.Kind(); got != emit.KindTag {
			t.Fatalf("Kind = %s, want %s", got, emit.KindTag)
		}
	})
}

func TestTag_ZeroValueUsable(t *testing.T) {
	t.Parallel()

	t.Run("the zero value carries no Key or Value", func(t *testing.T) {
		t.Parallel()
		var tg emit.Tag
		if tg.Key != "" || tg.Value != "" {
			t.Fatalf("zero Tag should be empty; got Key=%q Value=%q", tg.Key, tg.Value)
		}
	})
}
