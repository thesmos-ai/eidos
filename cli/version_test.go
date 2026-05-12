// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/plugin"
)

// stubPlugin is a minimal [plugin.Plugin] that records its Name
// only. Used across the cli command tests as a no-op universe entry.
type stubPlugin struct{ name string }

func (s stubPlugin) Name() string { return s.name }

// versionedStubPlugin wraps stubPlugin with a [plugin.Versioned]
// implementation so the Version command's "vX.Y.Z" rendering path
// is exercised.
type versionedStubPlugin struct {
	stubPlugin
	version string
}

func (v versionedStubPlugin) Version() string { return v.version }

// freshEnv returns an Env wired to bytes buffers so tests inspect
// what the command wrote without touching real stdio.
func freshEnv(t *testing.T, brand string) (env *cli.Env, stdout, stderr *bytes.Buffer) {
	t.Helper()
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	env = &cli.Env{
		Brand:   brand,
		Workdir: t.TempDir(),
		Stdout:  stdout,
		Stderr:  stderr,
	}
	return env, stdout, stderr
}

// TestVersionCommand_TextOutput covers the canonical text-format
// rendering: brand line, emit-contract line, sorted plugin list
// with enabled/disabled labels.
func TestVersionCommand_TextOutput(t *testing.T) {
	t.Parallel()

	t.Run("renders brand, emit-contract, and plugin list", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "acmegen")
		cmd := &cli.VersionCommand{Config: cli.VersionConfig{
			Plugins: []plugin.Plugin{
				versionedStubPlugin{stubPlugin: stubPlugin{name: "repogen"}, version: "1.2.0"},
				stubPlugin{name: "no-version-gen"},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want %d", code, cli.ExitOK)
		}
		out := stdout.String()
		if !strings.Contains(out, "acmegen\n") {
			t.Fatalf("expected brand line; got %q", out)
		}
		if !strings.Contains(out, "emit-contract:") {
			t.Fatalf("expected emit-contract line; got %q", out)
		}
		if !strings.Contains(out, "repogen") || !strings.Contains(out, "1.2.0") {
			t.Fatalf("expected repogen 1.2.0 line; got %q", out)
		}
		if !strings.Contains(out, "no-version-gen") || !strings.Contains(out, "dev") {
			t.Fatalf("expected unversioned plugin to render as dev; got %q", out)
		}
	})

	t.Run("empty plugin slice renders the (none) placeholder", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		cmd := &cli.VersionCommand{}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want %d", code, cli.ExitOK)
		}
		if !strings.Contains(stdout.String(), "plugins: (none)") {
			t.Fatalf("expected (none) placeholder; got %q", stdout.String())
		}
	})

	t.Run("config-disabled plugins render as disabled", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		off := false
		cfg := cli.DefaultConfig()
		cfg.Plugins = []cli.ConfigPlugin{
			{Name: "repogen"},
			{Name: "validation", Enabled: &off},
		}
		cmd := &cli.VersionCommand{Config: cli.VersionConfig{
			File: cfg,
			Plugins: []plugin.Plugin{
				stubPlugin{name: "repogen"},
				stubPlugin{name: "validation"},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want %d", code, cli.ExitOK)
		}
		out := stdout.String()
		if !strings.Contains(out, "repogen") || !strings.Contains(out, "(enabled)") {
			t.Fatalf("repogen should render enabled; got %q", out)
		}
		if !strings.Contains(out, "validation") || !strings.Contains(out, "(disabled)") {
			t.Fatalf("validation should render disabled; got %q", out)
		}
	})
}

// TestVersionCommand_JSONOutput covers the JSON rendering: one
// object on stdout with brand, emit_contract, plugins list of
// {name, version, enabled} records.
func TestVersionCommand_JSONOutput(t *testing.T) {
	t.Parallel()

	t.Run("renders a single JSON object with brand + plugins", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		cmd := &cli.VersionCommand{Config: cli.VersionConfig{
			Plugins: []plugin.Plugin{stubPlugin{name: "p1"}},
			Format:  cli.DiagFormatJSON,
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want %d", code, cli.ExitOK)
		}
		var out struct {
			Brand        string `json:"brand"`
			EmitContract string `json:"emit_contract"`
			Plugins      []struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Enabled bool   `json:"enabled"`
			} `json:"plugins"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
			t.Fatalf("not valid JSON: %v\n%q", err, stdout.String())
		}
		if out.Brand != "eidos" {
			t.Fatalf("Brand = %q, want eidos", out.Brand)
		}
		if len(out.Plugins) != 1 || out.Plugins[0].Name != "p1" {
			t.Fatalf("plugins payload mismatch; got %+v", out.Plugins)
		}
	})
}

// TestVersionCommand_RequiresBrand covers the env-validation guard:
// an empty Env.Brand exits with ExitUserError before any output.
func TestVersionCommand_RequiresBrand(t *testing.T) {
	t.Parallel()

	t.Run("empty brand exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env := &cli.Env{Brand: "", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
		cmd := &cli.VersionCommand{}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitUserError {
			t.Fatalf("Execute = %d, want ExitUserError", code)
		}
	})
}
