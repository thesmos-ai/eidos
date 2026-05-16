// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package monotonic_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/monotonic"
)

func TestMixin_Identity(t *testing.T) {
	t.Parallel()
	m := monotonic.Mixin()
	if m.Name != monotonic.Name {
		t.Fatalf("Mixin().Name = %q, want %q", m.Name, monotonic.Name)
	}
	if len(m.Params) != 0 {
		t.Fatalf("Mixin().Params = %v, want empty", m.Params)
	}
}
