// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugin"
)

// versionedStub implements [plugin.Versioned] for the contract test.
type versionedStub struct{ v string }

func (s versionedStub) Version() string { return s.v }

func TestVersioned_Contract(t *testing.T) {
	t.Parallel()

	t.Run("returns the configured version string", func(t *testing.T) {
		t.Parallel()
		var v plugin.Versioned = versionedStub{v: "1.2.3"}
		if got := v.Version(); got != "1.2.3" {
			t.Fatalf("Version() = %q, want %q", got, "1.2.3")
		}
	})

	t.Run("empty version is permitted and opts the plugin out of cache integration", func(t *testing.T) {
		t.Parallel()
		var v plugin.Versioned = versionedStub{}
		if v.Version() != "" {
			t.Fatalf("empty version should be returned verbatim")
		}
	})
}
