// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugin"
)

// stubFilenameProvider satisfies [plugin.FilenameProvider] for the
// contract-only tests below. The fixture returns a per-language
// slice so the multi-language and multi-output contract is
// exercised at the type level.
type stubFilenameProvider struct {
	goOutputs   []plugin.Output
	rustOutputs []plugin.Output
}

func (stubFilenameProvider) Name() string { return "stub-filename" }

func (s stubFilenameProvider) Outputs(lang string) []plugin.Output {
	switch lang {
	case "golang":
		return s.goOutputs
	case "rust":
		return s.rustOutputs
	}
	return nil
}

// TestOutput pins the [plugin.Output] value type's shape — the
// per-output declaration a [plugin.FilenameProvider] returns. Tag
// scopes the output within the plugin's namespace (empty for the
// plugin's primary output); Suffix is the per-source-basename
// trailer the Layout phase appends to compose the rendered
// output filename.
func TestOutput(t *testing.T) {
	t.Parallel()

	t.Run("zero value has empty Tag and empty Suffix", func(t *testing.T) {
		t.Parallel()
		var o plugin.Output
		if o.Tag != "" {
			t.Errorf("zero-value Tag = %q, want empty", o.Tag)
		}
		if o.Suffix != "" {
			t.Errorf("zero-value Suffix = %q, want empty", o.Suffix)
		}
	})

	t.Run("Tag and Suffix round-trip through struct literal", func(t *testing.T) {
		t.Parallel()
		o := plugin.Output{Tag: "test", Suffix: "_mock_test.go"}
		if o.Tag != "test" {
			t.Errorf("Tag = %q, want %q", o.Tag, "test")
		}
		if o.Suffix != "_mock_test.go" {
			t.Errorf("Suffix = %q, want %q", o.Suffix, "_mock_test.go")
		}
	})
}

// TestFilenameProvider_Contract pins the interface shape:
// implementing types satisfy [plugin.FilenameProvider] when they
// expose [plugin.Plugin] plus an [Outputs] method that takes the
// active backend's language and returns the ordered set of outputs
// the plugin emits for that language. The per-language return
// shape lets one plugin declare distinct outputs per backend —
// mirroring the [plugin.TemplateProvider.Templates] shape so the
// language-coupled surfaces stay symmetric.
func TestFilenameProvider_Contract(t *testing.T) {
	t.Parallel()

	t.Run("a type returning a per-language Outputs slice satisfies the interface", func(t *testing.T) {
		t.Parallel()
		var fp plugin.FilenameProvider = stubFilenameProvider{
			goOutputs:   []plugin.Output{{Suffix: "_repo.go"}},
			rustOutputs: []plugin.Output{{Suffix: "_repo.rs"}},
		}
		got := fp.Outputs("golang")
		if len(got) != 1 || got[0].Suffix != "_repo.go" || got[0].Tag != "" {
			t.Errorf("Outputs(golang) = %+v, want one entry with empty Tag and Suffix=_repo.go", got)
		}
		got = fp.Outputs("rust")
		if len(got) != 1 || got[0].Suffix != "_repo.rs" || got[0].Tag != "" {
			t.Errorf("Outputs(rust) = %+v, want one entry with empty Tag and Suffix=_repo.rs", got)
		}
		if fp.Name() != "stub-filename" {
			t.Fatalf("Name promoted from Plugin should round-trip; got %q", fp.Name())
		}
	})

	t.Run("a multi-output plugin returns the ordered output slice", func(t *testing.T) {
		t.Parallel()
		var fp plugin.FilenameProvider = stubFilenameProvider{
			goOutputs: []plugin.Output{
				{Suffix: "_enum.go"},
				{Tag: "test", Suffix: "_enum_test.go"},
			},
		}
		got := fp.Outputs("golang")
		if len(got) != 2 {
			t.Fatalf("Outputs(golang) returned %d entries, want 2", len(got))
		}
		if got[0].Tag != "" || got[0].Suffix != "_enum.go" {
			t.Errorf("entry[0] = %+v, want {Tag:%q, Suffix:%q}", got[0], "", "_enum.go")
		}
		if got[1].Tag != "test" || got[1].Suffix != "_enum_test.go" {
			t.Errorf("entry[1] = %+v, want {Tag:%q, Suffix:%q}", got[1], "test", "_enum_test.go")
		}
	})

	t.Run("an unsupported-language return is the empty-slice signal", func(t *testing.T) {
		t.Parallel()
		// A plugin that ships only go templates returns nil for
		// any other language — the framework treats that the same
		// as "no FilenameProvider declared" for the active backend.
		var fp plugin.FilenameProvider = stubFilenameProvider{
			goOutputs: []plugin.Output{{Suffix: "_repo.go"}},
		}
		if got := fp.Outputs("rust"); len(got) != 0 {
			t.Fatalf("Outputs(rust) = %+v, want empty (plugin ships no rust outputs)", got)
		}
	})

	t.Run("zero-value receiver returns empty for every language", func(t *testing.T) {
		t.Parallel()
		var fp plugin.FilenameProvider = stubFilenameProvider{}
		if got := fp.Outputs("golang"); len(got) != 0 {
			t.Fatalf("zero-value Outputs(golang) = %+v, want empty", got)
		}
	})
}
