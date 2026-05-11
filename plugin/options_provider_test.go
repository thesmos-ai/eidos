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

	t.Run("SetOptions decodes validated values into plugin state and returns nil on success", func(t *testing.T) {
		t.Parallel()
		op := &stubOptionsProvider{name: "stub"}
		o := opt.New(op.OptionsSchema(), map[string]string{"output": "internal/users"})
		if err := op.SetOptions(o); err != nil {
			t.Fatalf("SetOptions should succeed for valid input; got %v", err)
		}
	})

	t.Run("SetOptions returns the validation error for invalid input", func(t *testing.T) {
		t.Parallel()
		op := &stubOptionsProvider{name: "stub"}
		// Required field 'output' is missing → Decode returns the
		// error and SetOptions surfaces it.
		o := opt.New(op.OptionsSchema(), map[string]string{})
		if err := op.SetOptions(o); err == nil {
			t.Fatalf("SetOptions should return an error for missing required input")
		}
	})
}
