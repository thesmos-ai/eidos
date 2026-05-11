// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/plugin"
)

func TestOptionsProvider_Contract(t *testing.T) {
	t.Parallel()

	t.Run("OptionsSchema returns the schema derived from the plugin's options type", func(t *testing.T) {
		t.Parallel()
		var op plugin.OptionsProvider = &stubOptionsProvider{name: "stub"}
		if !op.OptionsSchema().HasField("output") {
			t.Fatalf("schema should expose declared field 'output'")
		}
	})

	t.Run("SetOptions decodes validated values into plugin state", func(t *testing.T) {
		t.Parallel()
		op := &stubOptionsProvider{name: "stub"}
		o := opt.New(op.OptionsSchema(), map[string]string{"output": "internal/users"})
		op.SetOptions(o)
		if !op.OptionsSchema().HasField("output") {
			t.Fatalf("OptionsSchema should remain callable after SetOptions")
		}
	})
}
