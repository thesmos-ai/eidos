// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestEnumVariant_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindEnumVariant", func(t *testing.T) {
		t.Parallel()
		var v emit.EnumVariant
		if v.Kind() != emit.KindEnumVariant {
			t.Fatalf("Kind = %s, want %s", v.Kind(), emit.KindEnumVariant)
		}
	})
}
