// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
)

func TestCapabilityProvider_Contract(t *testing.T) {
	t.Parallel()

	t.Run("returns priority and capability lists from the implementing plugin", func(t *testing.T) {
		t.Parallel()
		var c plugin.CapabilityProvider = &stubCapability{
			name:     "stub",
			priority: priority.GeneratorComposition,
			provides: []string{"a"},
			requires: []string{"b", "c"},
		}
		if c.Priority() != priority.GeneratorComposition {
			t.Fatalf("Priority = %d, want %d", c.Priority(), priority.GeneratorComposition)
		}
		if !slices.Equal(c.Provides(), []string{"a"}) {
			t.Fatalf("Provides = %v, want [a]", c.Provides())
		}
		if !slices.Equal(c.Requires(), []string{"b", "c"}) {
			t.Fatalf("Requires = %v, want [b c]", c.Requires())
		}
	})

	t.Run("Plugin role is composable with CapabilityProvider", func(t *testing.T) {
		t.Parallel()
		var p plugin.Plugin = &stubCapability{name: "stub"}
		if _, ok := p.(plugin.CapabilityProvider); !ok {
			t.Fatalf("stubCapability should also satisfy CapabilityProvider")
		}
	})
}
