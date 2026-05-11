// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugin"
)

func TestPlugin_NameContract(t *testing.T) {
	t.Parallel()

	t.Run("a value implementing Name satisfies the Plugin interface", func(t *testing.T) {
		t.Parallel()
		var p plugin.Plugin = &stubFrontend{name: "stub"}
		if p.Name() != "stub" {
			t.Fatalf("Name = %q, want stub", p.Name())
		}
	})
}
