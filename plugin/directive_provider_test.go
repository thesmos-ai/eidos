// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/plugin"
)

func TestDirectiveProvider_Contract(t *testing.T) {
	t.Parallel()

	t.Run("Directives returns the schemas the plugin contributes", func(t *testing.T) {
		t.Parallel()
		schemas := []directive.Schema{
			directive.NewSchema("repo").Build(),
			directive.NewSchema("mock").Build(),
		}
		var dp plugin.DirectiveProvider = &stubDirectiveProvider{name: "stub", schemas: schemas}
		got := dp.Directives()
		if len(got) != 2 {
			t.Fatalf("Directives = %d schemas, want 2", len(got))
		}
	})

	t.Run("Directives returns nil for a plugin that contributes nothing", func(t *testing.T) {
		t.Parallel()
		var dp plugin.DirectiveProvider = &stubDirectiveProvider{name: "stub"}
		if dp.Directives() != nil {
			t.Fatalf("empty DirectiveProvider should return nil")
		}
	})
}
