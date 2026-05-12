// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/core/directive"
)

// TestLoadConfig_Defaults covers the empty-path entry point: an
// empty string returns a *Config seeded with the documented
// defaults, never touching disk.
func TestLoadConfig_Defaults(t *testing.T) {
	t.Parallel()

	t.Run("empty path returns DefaultConfig", func(t *testing.T) {
		t.Parallel()
		c, err := cli.LoadConfig("")
		if err != nil {
			t.Fatalf("LoadConfig(\"\") returned %v", err)
		}
		if c.Version != cli.ConfigVersion {
			t.Fatalf("default Version = %d, want %d", c.Version, cli.ConfigVersion)
		}
		if c.Directives.Prefix != directive.DefaultPrefix {
			t.Fatalf("default Directives.Prefix = %q, want %q", c.Directives.Prefix, directive.DefaultPrefix)
		}
		if c.Sink.Kind != "disk" {
			t.Fatalf("default Sink.Kind = %q, want %q", c.Sink.Kind, "disk")
		}
	})

	t.Run("DefaultConfig has empty Plugins / Sources slices but is non-nil", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		if c.Sources == nil || len(c.Sources) != 0 {
			t.Fatalf("Sources should be non-nil empty slice; got %v", c.Sources)
		}
	})
}

// TestLoadConfig_YAML covers the on-disk happy path: a valid YAML
// file populates the *Config field-by-field, defaults fill in for
// omitted fields.
func TestLoadConfig_YAML(t *testing.T) {
	t.Parallel()

	t.Run("populated YAML hydrates every field", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".eidos.yaml")
		body := []byte(`version: 1
sources:
  - frontend: golang
    patterns: ["./..."]
plugins:
  - name: repogen
    options:
      output_package: repo
  - name: validation
    enabled: false
sink:
  kind: disk
cache:
  enabled: true
  dir: ./build/cache
manifest:
  path: ./.eidos/manifest.json
directives:
  prefix: gen
parallel:
  - annotator
envelope:
  header_prefix: ["// Copyright X"]
verbose: true
`)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		c, err := cli.LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if len(c.Sources) != 1 || c.Sources[0].Frontend != "golang" {
			t.Fatalf("Sources not populated; got %+v", c.Sources)
		}
		if len(c.Plugins) != 2 || c.Plugins[0].Name != "repogen" {
			t.Fatalf("Plugins not populated; got %+v", c.Plugins)
		}
		if c.Plugins[0].IsEnabled() != true {
			t.Fatalf("repogen should default to enabled")
		}
		if c.Plugins[1].IsEnabled() != false {
			t.Fatalf("validation should be disabled per the file")
		}
		if !c.Verbose {
			t.Fatalf("Verbose should be true per the file")
		}
		if len(c.Envelope.HeaderPrefix) != 1 {
			t.Fatalf("HeaderPrefix not populated; got %v", c.Envelope.HeaderPrefix)
		}
		if c.Cache.Dir != "./build/cache" {
			t.Fatalf("Cache.Dir = %q, want %q", c.Cache.Dir, "./build/cache")
		}
	})

	t.Run("missing file returns ConfigError", func(t *testing.T) {
		t.Parallel()
		_, err := cli.LoadConfig(filepath.Join(t.TempDir(), "missing.yaml"))
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *ConfigError; got %T %v", err, err)
		}
	})

	t.Run("unsupported version is rejected", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".eidos.yaml")
		if err := os.WriteFile(path, []byte("version: 999\n"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		_, err := cli.LoadConfig(path)
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *ConfigError; got %T %v", err, err)
		}
	})

	t.Run("malformed YAML surfaces as ConfigError", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".eidos.yaml")
		if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		_, err := cli.LoadConfig(path)
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *ConfigError; got %T %v", err, err)
		}
	})

	t.Run("unknown sink kind is rejected", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".eidos.yaml")
		if err := os.WriteFile(path, []byte("version: 1\nsink:\n  kind: invalid\n"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		_, err := cli.LoadConfig(path)
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *ConfigError; got %T %v", err, err)
		}
	})
}

// TestConfigPlugin_IsEnabled covers the ConfigPlugin.IsEnabled
// default: omitted Enabled field treats the plugin as enabled;
// an explicit false disables it.
func TestConfigPlugin_IsEnabled(t *testing.T) {
	t.Parallel()

	t.Run("nil Enabled defaults to true", func(t *testing.T) {
		t.Parallel()
		p := cli.ConfigPlugin{Name: "repogen"}
		if !p.IsEnabled() {
			t.Fatalf("ConfigPlugin without Enabled should default to enabled")
		}
	})

	t.Run("explicit false disables", func(t *testing.T) {
		t.Parallel()
		off := false
		p := cli.ConfigPlugin{Name: "repogen", Enabled: &off}
		if p.IsEnabled() {
			t.Fatalf("ConfigPlugin with Enabled=false should report disabled")
		}
	})
}

// TestConfigCache_IsEnabled mirrors [TestConfigPlugin_IsEnabled]
// for the cache toggle.
func TestConfigCache_IsEnabled(t *testing.T) {
	t.Parallel()

	t.Run("nil Enabled defaults to true", func(t *testing.T) {
		t.Parallel()
		c := cli.ConfigCache{}
		if !c.IsEnabled() {
			t.Fatalf("ConfigCache without Enabled should default to enabled")
		}
	})

	t.Run("explicit false disables", func(t *testing.T) {
		t.Parallel()
		off := false
		c := cli.ConfigCache{Enabled: &off}
		if c.IsEnabled() {
			t.Fatalf("ConfigCache with Enabled=false should report disabled")
		}
	})
}

// TestLoadConfig_ValidationFailures covers each validation-failure
// arm not already exercised by TestLoadConfig_YAML: a source
// missing its frontend name, a plugin missing its name. Each
// surfaces as a *ConfigError naming the offending field.
func TestLoadConfig_ValidationFailures(t *testing.T) {
	t.Parallel()

	t.Run("source without frontend is rejected", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".eidos.yaml")
		body := []byte("version: 1\nsources:\n  - patterns: [\"./...\"]\n")
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		_, err := cli.LoadConfig(path)
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *ConfigError; got %T %v", err, err)
		}
	})

	t.Run("plugin without name is rejected", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".eidos.yaml")
		body := []byte("version: 1\nplugins:\n  - enabled: true\n")
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		_, err := cli.LoadConfig(path)
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *ConfigError; got %T %v", err, err)
		}
	})

	t.Run("default values fill in for omitted fields", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".eidos.yaml")
		// File omits version, directives.prefix, sink.kind — all
		// should be filled by validateConfig.
		if err := os.WriteFile(path, []byte("verbose: true\n"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		c, err := cli.LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if c.Version != cli.ConfigVersion {
			t.Fatalf("Version default = %d, want %d", c.Version, cli.ConfigVersion)
		}
		if c.Directives.Prefix != "gen" {
			t.Fatalf("Directives.Prefix default = %q, want %q", c.Directives.Prefix, "gen")
		}
		if c.Sink.Kind != "disk" {
			t.Fatalf("Sink.Kind default = %q, want %q", c.Sink.Kind, "disk")
		}
	})
}

// TestDiscoverConfig covers the walk-up discovery routine: it
// finds the config file in a parent directory, stops at the
// filesystem root.
func TestDiscoverConfig(t *testing.T) {
	t.Parallel()

	t.Run("walks up to find a config file in a parent directory", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		nested := filepath.Join(root, "a", "b", "c")
		if err := os.MkdirAll(nested, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		cfgPath := filepath.Join(root, ".eidos.yaml")
		if err := os.WriteFile(cfgPath, []byte("version: 1\n"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		got, ok := cli.DiscoverConfig(nested, ".eidos.yaml")
		if !ok {
			t.Fatalf("DiscoverConfig should have found the config")
		}
		gotAbs, _ := filepath.Abs(got)
		wantAbs, _ := filepath.Abs(cfgPath)
		if gotAbs != wantAbs {
			t.Fatalf("DiscoverConfig got %q, want %q", gotAbs, wantAbs)
		}
	})

	t.Run("returns false when no config is found up to the root", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		nested := filepath.Join(root, "a", "b", "c")
		if err := os.MkdirAll(nested, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		_, ok := cli.DiscoverConfig(nested, ".nonexistent.yaml")
		if ok {
			t.Fatalf("DiscoverConfig should have returned false for a missing config")
		}
	})
}
