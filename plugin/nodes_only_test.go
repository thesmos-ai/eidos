// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugin"
)

// stubNodesOnly is a stub generator that implements NodesOnly. The
// declared value flips between tests via the v field.
type stubNodesOnly struct {
	name string
	v    bool
}

func (s *stubNodesOnly) Name() string                            { return s.name }
func (*stubNodesOnly) Generate(_ *plugin.GeneratorContext) error { return nil }
func (s *stubNodesOnly) NodesOnly() bool                         { return s.v }

func TestNodesOnly_Contract(t *testing.T) {
	t.Parallel()

	t.Run("a value implementing NodesOnly satisfies the interface", func(t *testing.T) {
		t.Parallel()
		var no plugin.NodesOnly = &stubNodesOnly{name: "g", v: true}
		if !no.NodesOnly() {
			t.Fatalf("NodesOnly should return the declared value")
		}
	})

	t.Run("a value returning false still satisfies the interface", func(t *testing.T) {
		t.Parallel()
		var no plugin.NodesOnly = &stubNodesOnly{name: "g", v: false}
		if no.NodesOnly() {
			t.Fatalf("NodesOnly should return the declared value")
		}
	})
}
