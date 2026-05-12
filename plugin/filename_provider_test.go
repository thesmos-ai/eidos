// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugin"
)

// stubFilenameProvider satisfies [plugin.FilenameProvider] for the
// contract-only test below.
type stubFilenameProvider struct{ suffix string }

func (stubFilenameProvider) Name() string             { return "stub-filename" }
func (s stubFilenameProvider) FilenameSuffix() string { return s.suffix }

// TestFilenameProvider_Contract pins the interface shape: implementing
// types satisfy [plugin.FilenameProvider] when they expose
// [plugin.Plugin] plus a [FilenameSuffix] method.
func TestFilenameProvider_Contract(t *testing.T) {
	t.Parallel()

	t.Run("a type returning the suffix satisfies the interface", func(t *testing.T) {
		t.Parallel()
		var fp plugin.FilenameProvider = stubFilenameProvider{suffix: "_repo.go"}
		if fp.FilenameSuffix() != "_repo.go" {
			t.Fatalf("FilenameSuffix = %q, want %q", fp.FilenameSuffix(), "_repo.go")
		}
		if fp.Name() != "stub-filename" {
			t.Fatalf("Name promoted from Plugin should round-trip; got %q", fp.Name())
		}
	})

	t.Run("an empty-suffix return is the framework default signal", func(t *testing.T) {
		t.Parallel()
		var fp plugin.FilenameProvider = stubFilenameProvider{}
		if fp.FilenameSuffix() != "" {
			t.Fatalf("zero-value FilenameSuffix should be the empty string; got %q", fp.FilenameSuffix())
		}
	})
}
