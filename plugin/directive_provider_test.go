// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/plugin"
)

// stubDirectiveProvider satisfies [plugin.Plugin] and
// [plugin.DirectiveProvider] for the auto-collection test
// path — used both here for contract verification and by pipeline
// tests that exercise the Build-time collection loop.
type stubDirectiveProvider struct {
	name    string
	schemas []directive.Schema
}

func (s *stubDirectiveProvider) Name() string                   { return s.name }
func (s *stubDirectiveProvider) Directives() []directive.Schema { return s.schemas }

// TestDirectiveProviderInterface covers the contract surface: the
// interface is reachable, satisfied by concrete plugins, and
// Directives returns the declared schema list verbatim.
func TestDirectiveProviderInterface(t *testing.T) {
	t.Parallel()

	t.Run("an implementing plugin satisfies plugin.DirectiveProvider", func(t *testing.T) {
		t.Parallel()
		schema := directive.NewSchema("test").Build()
		var dp plugin.DirectiveProvider = &stubDirectiveProvider{
			name:    "stub",
			schemas: []directive.Schema{schema},
		}
		got := dp.Directives()
		if len(got) != 1 || got[0].Name != "test" {
			t.Fatalf("Directives = %+v; want one schema named %q", got, "test")
		}
		if dp.Name() != "stub" {
			t.Fatalf("Name = %q, want %q", dp.Name(), "stub")
		}
	})

	t.Run("a plugin with no schemas still satisfies the interface", func(t *testing.T) {
		t.Parallel()
		var dp plugin.DirectiveProvider = &stubDirectiveProvider{name: "empty"}
		if got := dp.Directives(); got != nil {
			t.Fatalf("empty Directives should return nil; got %+v", got)
		}
	})
}
