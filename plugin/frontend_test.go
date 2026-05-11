// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

func TestFrontend_Contract(t *testing.T) {
	t.Parallel()

	t.Run("Load populates the source store with at least one node", func(t *testing.T) {
		t.Parallel()
		var fe plugin.Frontend = &stubFrontend{name: "stub"}
		s := store.New()
		assertNoError(t, fe.Load("input", s, diag.New()))
		if s.Nodes().Structs().Len() == 0 {
			t.Fatalf("Load should have populated the source store")
		}
	})

	t.Run("Frontend satisfies Plugin via name", func(t *testing.T) {
		t.Parallel()
		var p plugin.Plugin = &stubFrontend{name: "go"}
		if p.Name() != "go" {
			t.Fatalf("Name = %q, want go", p.Name())
		}
	})
}
