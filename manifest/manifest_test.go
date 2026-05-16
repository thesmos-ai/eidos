// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/manifest"
)

// TestPluginAttribution_String pins the cross-tool naming
// convention: an untagged attribution renders as the bare plugin
// name; a tagged attribution renders as `<plugin>:<tag>`. The
// CLI `-o <plugin>:<tag>=<path>` form established the canonical
// rendering, and every human-readable surface (diagnostics, log
// lines, explain output) honours the same shape so a reader of a
// multi-plugin pipeline can tell whose tag a message refers to.
func TestPluginAttribution_String(t *testing.T) {
	t.Parallel()

	t.Run("untagged attribution renders as the bare plugin name", func(t *testing.T) {
		t.Parallel()
		a := manifest.PluginAttribution{Name: "enum"}
		if got, want := a.String(), "enum"; got != want {
			t.Errorf("String = %q, want %q", got, want)
		}
	})

	t.Run("tagged attribution renders as `<plugin>:<tag>`", func(t *testing.T) {
		t.Parallel()
		a := manifest.PluginAttribution{Name: "enum", OutputTag: "test"}
		if got, want := a.String(), "enum:test"; got != want {
			t.Errorf("String = %q, want %q", got, want)
		}
	})

	t.Run("zero value renders as the empty string", func(t *testing.T) {
		t.Parallel()
		var a manifest.PluginAttribution
		if got := a.String(); got != "" {
			t.Errorf("zero-value String = %q, want empty", got)
		}
	})
}

// TestPluginAttribution_MarshalJSON pins the per-plugin manifest
// serialisation: an untagged attribution serialises as a bare
// JSON string (preserving byte-stable parity with pre-multi-output
// manifests written before the output_tag field landed); a tagged
// attribution serialises as a JSON object with `name` and
// `output_tag` fields.
func TestPluginAttribution_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("untagged attribution serialises as a bare string", func(t *testing.T) {
		t.Parallel()
		a := manifest.PluginAttribution{Name: "enum"}
		got, err := json.Marshal(a)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if want := `"enum"`; string(got) != want {
			t.Errorf("Marshal = %s, want %s", got, want)
		}
	})

	t.Run("tagged attribution serialises as {name, output_tag}", func(t *testing.T) {
		t.Parallel()
		a := manifest.PluginAttribution{Name: "enum", OutputTag: "test"}
		got, err := json.Marshal(a)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if want := `{"name":"enum","output_tag":"test"}`; string(got) != want {
			t.Errorf("Marshal = %s, want %s", got, want)
		}
	})

	t.Run("untagged attribution round-trips through string-form unmarshal", func(t *testing.T) {
		t.Parallel()
		var a manifest.PluginAttribution
		if err := json.Unmarshal([]byte(`"enum"`), &a); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if a.Name != "enum" || a.OutputTag != "" {
			t.Errorf("Unmarshal = %+v, want {Name:enum OutputTag:}", a)
		}
	})

	t.Run("tagged attribution round-trips through object-form unmarshal", func(t *testing.T) {
		t.Parallel()
		var a manifest.PluginAttribution
		if err := json.Unmarshal([]byte(`{"name":"enum","output_tag":"test"}`), &a); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if a.Name != "enum" || a.OutputTag != "test" {
			t.Errorf("Unmarshal = %+v, want {Name:enum OutputTag:test}", a)
		}
	})

	t.Run("type-mismatched bare-string input surfaces a wrapped error", func(t *testing.T) {
		t.Parallel()
		// Direct UnmarshalJSON invocation bypasses the outer
		// scanner's validation so we exercise the bare-string
		// branch's error wrap. A leading `"` routes through
		// `json.Unmarshal(data, &a.Name)`; truncated quote-bytes
		// fail at that step.
		var a manifest.PluginAttribution
		err := a.UnmarshalJSON([]byte(`"unterminated`))
		if err == nil {
			t.Fatalf("expected unmarshal error for malformed bare-string input")
		}
		if !strings.Contains(err.Error(), "manifest:") {
			t.Errorf("error should carry manifest prefix; got %q", err.Error())
		}
	})

	t.Run("type-mismatched object input surfaces a wrapped error", func(t *testing.T) {
		t.Parallel()
		// The outer scanner accepts the object form (valid JSON);
		// the inner unmarshal into the typed alias fails because
		// `name` is declared as a string but the payload carries
		// an integer.
		var a manifest.PluginAttribution
		err := json.Unmarshal([]byte(`{"name":123}`), &a)
		if err == nil {
			t.Fatalf("expected unmarshal error for type-mismatched object input")
		}
		if !strings.Contains(err.Error(), "manifest:") {
			t.Errorf("error should carry manifest prefix; got %q", err.Error())
		}
	})
}

// TestOutput_Plugins_ByteStability pins the byte-stability contract
// for [manifest.Output.Plugins]: an output whose plugins all carry
// empty OutputTag values marshals to the legacy bare-string JSON
// array, matching pre-multi-output manifests. Any plugin with a
// non-empty OutputTag flips that plugin's element to the object
// form.
func TestOutput_Plugins_ByteStability(t *testing.T) {
	t.Parallel()

	t.Run("untagged Plugins marshal as a bare string array", func(t *testing.T) {
		t.Parallel()
		out := manifest.Output{
			Target: targetAt("a", "x.go"),
			Plugins: []manifest.PluginAttribution{
				{Name: "enum"},
				{Name: "weaver"},
			},
			Hash: "sha256:abc",
		}
		got, err := json.Marshal(out)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		// Spot-check the Plugins fragment — the precise field
		// ordering of Output is enforced by other tests.
		if want := `"plugins":["enum","weaver"]`; !contains(string(got), want) {
			t.Errorf("Marshal plugins fragment missing %s; got:\n%s", want, got)
		}
	})

	t.Run("mixed tagged + untagged Plugins marshal as a polymorphic array", func(t *testing.T) {
		t.Parallel()
		out := manifest.Output{
			Target: targetAt("a", "x_test.go"),
			Plugins: []manifest.PluginAttribution{
				{Name: "enum", OutputTag: "test"},
				{Name: "weaver"},
			},
			Hash: "sha256:abc",
		}
		got, err := json.Marshal(out)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		want := `"plugins":[{"name":"enum","output_tag":"test"},"weaver"]`
		if !contains(string(got), want) {
			t.Errorf("Marshal plugins fragment missing %s; got:\n%s", want, got)
		}
	})
}

// contains is the test-local strings.Contains alias used by the
// byte-stability assertions; importing strings just for this one
// site clutters the manifest test file's imports.
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a Manifest stamped with current Version and supplied RunID", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		if m.Version != manifest.Version {
			t.Fatalf("Version = %d, want %d", m.Version, manifest.Version)
		}
		if m.RunID != "run-1" {
			t.Fatalf("RunID = %q, want run-1", m.RunID)
		}
		if len(m.Outputs) != 0 {
			t.Fatalf("new Manifest should have no outputs; got %d", len(m.Outputs))
		}
		if m.Brand != "" {
			t.Fatalf("Brand should default to empty for the caller to stamp; got %q", m.Brand)
		}
	})

	t.Run("Brand is a stamped field callers populate after construction", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-2")
		m.Brand = "acmegen"
		if m.Brand != "acmegen" {
			t.Fatalf("Brand mutation didn't stick; got %q", m.Brand)
		}
	})
}

func TestManifest_Add(t *testing.T) {
	t.Parallel()

	t.Run("appends an output entry", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		m.Add(
			manifest.Output{
				Target:  targetAt("a", "b.go"),
				Plugins: []manifest.PluginAttribution{{Name: "p"}},
				Hash:    "sha256:x",
			},
		)
		if len(m.Outputs) != 1 {
			t.Fatalf("Add should append; Outputs=%d", len(m.Outputs))
		}
	})
}

func TestManifest_Targets(t *testing.T) {
	t.Parallel()

	t.Run("returns the target list in append order", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		m.Add(manifest.Output{Target: targetAt("a", "first.go")})
		m.Add(manifest.Output{Target: targetAt("a", "second.go")})
		got := m.Targets()
		want := []string{"first.go", "second.go"}
		gotNames := []string{got[0].Filename, got[1].Filename}
		if !slices.Equal(gotNames, want) {
			t.Fatalf("Targets order mismatch: %v", gotNames)
		}
	})

	t.Run("returns an empty slice for an empty manifest", func(t *testing.T) {
		t.Parallel()
		got := manifest.New("run-1").Targets()
		if len(got) != 0 {
			t.Fatalf("empty manifest should yield no targets; got %d", len(got))
		}
	})
}

func TestManifest_HasTarget(t *testing.T) {
	t.Parallel()

	t.Run("returns true for a recorded target", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		m.Add(manifest.Output{Target: targetAt("a", "b.go")})
		if !m.HasTarget(targetAt("a", "b.go")) {
			t.Fatalf("HasTarget should return true for recorded target")
		}
	})

	t.Run("returns false for an unknown target", func(t *testing.T) {
		t.Parallel()
		m := manifest.New("run-1")
		if m.HasTarget(targetAt("a", "missing.go")) {
			t.Fatalf("HasTarget should return false for unknown target")
		}
	})
}
