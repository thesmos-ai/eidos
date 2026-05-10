// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestEmbed_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindEmbed", func(t *testing.T) {
		t.Parallel()
		var e emit.Embed
		if e.Kind() != emit.KindEmbed {
			t.Fatalf("Kind = %s, want %s", e.Kind(), emit.KindEmbed)
		}
	})
}
