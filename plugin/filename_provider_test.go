// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugin"
)

// stubFilenameProvider satisfies [plugin.FilenameProvider] for the
// contract-only test below. The test fixture returns a per-language
// suffix so the multi-language contract is exercised at the type
// level.
type stubFilenameProvider struct {
	goSuffix   string
	rustSuffix string
}

func (stubFilenameProvider) Name() string { return "stub-filename" }

func (s stubFilenameProvider) FilenameSuffix(lang string) string {
	switch lang {
	case "golang":
		return s.goSuffix
	case "rust":
		return s.rustSuffix
	}
	return ""
}

// TestFilenameProvider_Contract pins the interface shape: implementing
// types satisfy [plugin.FilenameProvider] when they expose
// [plugin.Plugin] plus a [FilenameSuffix] method that takes the
// active backend's language. The method's per-language return
// values let one plugin declare distinct suffixes per backend
// — mirroring the [plugin.TemplateProvider.Templates] shape so
// the language-coupled surfaces stay symmetric.
func TestFilenameProvider_Contract(t *testing.T) {
	t.Parallel()

	t.Run("a type returning a per-language suffix satisfies the interface", func(t *testing.T) {
		t.Parallel()
		var fp plugin.FilenameProvider = stubFilenameProvider{
			goSuffix:   "_repo.go",
			rustSuffix: "_repo.rs",
		}
		if got, want := fp.FilenameSuffix("golang"), "_repo.go"; got != want {
			t.Errorf("FilenameSuffix(golang) = %q, want %q", got, want)
		}
		if got, want := fp.FilenameSuffix("rust"), "_repo.rs"; got != want {
			t.Errorf("FilenameSuffix(rust) = %q, want %q", got, want)
		}
		if fp.Name() != "stub-filename" {
			t.Fatalf("Name promoted from Plugin should round-trip; got %q", fp.Name())
		}
	})

	t.Run("an unsupported-language return is the empty-string signal", func(t *testing.T) {
		t.Parallel()
		// A plugin that ships only go templates returns the
		// empty string for any other language — the framework
		// treats that the same as "no FilenameProvider declared"
		// for the active backend.
		var fp plugin.FilenameProvider = stubFilenameProvider{goSuffix: "_repo.go"}
		if got := fp.FilenameSuffix("rust"); got != "" {
			t.Fatalf("FilenameSuffix(rust) = %q, want empty (plugin ships no rust suffix)", got)
		}
	})

	t.Run("zero-value receiver returns empty for every language", func(t *testing.T) {
		t.Parallel()
		var fp plugin.FilenameProvider = stubFilenameProvider{}
		if got := fp.FilenameSuffix("golang"); got != "" {
			t.Fatalf("zero-value FilenameSuffix(golang) = %q, want empty", got)
		}
	})
}
