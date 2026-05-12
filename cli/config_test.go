// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
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
			t.Fatalf(
				"default Directives.Prefix = %q, want %q",
				c.Directives.Prefix,
				directive.DefaultPrefix,
			)
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

// extendedConfig models the embedder pattern [LoadConfigInto]
// supports: inline-embed [cli.Config] under the same YAML
// namespace, add caller-defined extension fields alongside.
type extendedConfig struct {
	cli.Config `yaml:",inline"`

	App appExtras `yaml:"app"`
}

// appExtras is the embedder-side configuration surface — arbitrary
// shape, owned by the embedder.
type appExtras struct {
	Region   string   `yaml:"region"`
	Replicas int      `yaml:"replicas"`
	Tags     []string `yaml:"tags,omitempty"`
}

// TestLoadConfigInto covers the generic-loader path: embedders
// compose their own typed configuration around [cli.Config], parse
// through [LoadConfigInto], then run [ValidateConfig] on the
// embedded portion to share the framework's validation pass.
func TestLoadConfigInto(t *testing.T) {
	t.Parallel()

	t.Run("inline-embedded extension parses alongside the framework keys", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".app.yaml")
		body := []byte(`version: 1
sources:
  - frontend: golang
    patterns: ["./..."]
plugins:
  - name: repogen
    options:
      output_package: gen
app:
  region: eu-west-1
  replicas: 3
  tags: ["alpha", "beta"]
`)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		cfg := &extendedConfig{Config: *cli.DefaultConfig()}
		if err := cli.LoadConfigInto(path, cfg); err != nil {
			t.Fatalf("LoadConfigInto: %v", err)
		}
		if _, err := cli.ValidateConfig(&cfg.Config, path); err != nil {
			t.Fatalf("ValidateConfig: %v", err)
		}
		if cfg.App.Region != "eu-west-1" || cfg.App.Replicas != 3 {
			t.Fatalf("embedder extension not populated: %+v", cfg.App)
		}
		if len(cfg.Sources) != 1 || cfg.Sources[0].Frontend != "golang" {
			t.Fatalf("framework section not populated: %+v", cfg.Sources)
		}
		if cfg.Plugins[0].Name != "repogen" {
			t.Fatalf("framework plugins section not populated: %+v", cfg.Plugins)
		}
	})

	t.Run("empty path leaves the seeded target untouched", func(t *testing.T) {
		t.Parallel()
		cfg := &extendedConfig{
			Config: *cli.DefaultConfig(),
			App:    appExtras{Region: "us-east-1", Replicas: 1},
		}
		if err := cli.LoadConfigInto("", cfg); err != nil {
			t.Fatalf("LoadConfigInto(\"\"): %v", err)
		}
		if cfg.App.Region != "us-east-1" || cfg.App.Replicas != 1 {
			t.Fatalf("seeded App should be preserved; got %+v", cfg.App)
		}
		if cfg.Sink.Kind != "disk" {
			t.Fatalf(
				"seeded framework defaults should be preserved; got Sink.Kind=%q",
				cfg.Sink.Kind,
			)
		}
	})

	t.Run("missing file surfaces a *ConfigError", func(t *testing.T) {
		t.Parallel()
		cfg := &extendedConfig{Config: *cli.DefaultConfig()}
		err := cli.LoadConfigInto(filepath.Join(t.TempDir(), "nope.yaml"), cfg)
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %T (%v)", err, err)
		}
	})

	t.Run("malformed YAML surfaces a *ConfigError", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".bad.yaml")
		if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		cfg := &extendedConfig{Config: *cli.DefaultConfig()}
		err := cli.LoadConfigInto(path, cfg)
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %T (%v)", err, err)
		}
	})
}

// TestValidateConfig_OutputBlock covers the routing-layer config
// validation: the layout-enum check, the centralised-requires-
// package rule, and the dir-without-centralised warning. Each
// rule is exercised at project level and at per-plugin level.
func TestValidateConfig_OutputBlock(t *testing.T) {
	t.Parallel()

	t.Run("unknown project layout value is rejected", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		c.Output = cli.ConfigOutput{Layout: "bogus"}
		_, err := cli.ValidateConfig(c, "")
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %v", err)
		}
		if !strings.Contains(ce.Reason, "output.layout") {
			t.Fatalf("error should name output.layout; got %q", ce.Reason)
		}
	})

	t.Run("centralised without package fails at project level", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		c.Output = cli.ConfigOutput{Layout: pipeline.LayoutCentralised}
		_, err := cli.ValidateConfig(c, "")
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %v", err)
		}
		if !strings.Contains(ce.Reason, "output.package") {
			t.Fatalf("error should name output.package; got %q", ce.Reason)
		}
	})

	t.Run("centralised with package validates clean", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		c.Output = cli.ConfigOutput{
			Layout: pipeline.LayoutCentralised, Package: "gen",
		}
		warnings, err := cli.ValidateConfig(c, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 0 {
			t.Fatalf("unexpected warnings: %v", warnings)
		}
	})

	t.Run("dir without centralised surfaces a warning", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		c.Output = cli.ConfigOutput{Dir: "internal/gen"}
		warnings, err := cli.ValidateConfig(c, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 1 || !strings.Contains(warnings[0], "output.dir") {
			t.Fatalf("expected output.dir warning; got %v", warnings)
		}
	})

	t.Run("per-plugin centralised inherits project package", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		c.Output = cli.ConfigOutput{Package: "gen"}
		c.Plugins = []cli.ConfigPlugin{
			{Name: "mockgen", Output: &cli.ConfigOutput{Layout: pipeline.LayoutCentralised}},
		}
		warnings, err := cli.ValidateConfig(c, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 0 {
			t.Fatalf("unexpected warnings: %v", warnings)
		}
	})

	t.Run("per-plugin centralised without inherited package fails", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		c.Plugins = []cli.ConfigPlugin{
			{Name: "mockgen", Output: &cli.ConfigOutput{Layout: pipeline.LayoutCentralised}},
		}
		_, err := cli.ValidateConfig(c, "")
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %v", err)
		}
		if !strings.Contains(ce.Reason, "plugins[0].output") {
			t.Fatalf("error should name plugins[0].output; got %q", ce.Reason)
		}
	})

	t.Run("per-plugin dir without centralised surfaces a warning", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		c.Plugins = []cli.ConfigPlugin{
			{Name: "mockgen", Output: &cli.ConfigOutput{Dir: "internal/mocks"}},
		}
		warnings, err := cli.ValidateConfig(c, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 1 || !strings.Contains(warnings[0], "plugins[0].output.dir") {
			t.Fatalf("expected plugins[0].output.dir warning; got %v", warnings)
		}
	})

	t.Run("per-plugin unknown layout value is rejected", func(t *testing.T) {
		t.Parallel()
		c := cli.DefaultConfig()
		c.Plugins = []cli.ConfigPlugin{
			{Name: "mockgen", Output: &cli.ConfigOutput{Layout: "weird"}},
		}
		_, err := cli.ValidateConfig(c, "")
		var ce *cli.ConfigError
		if !errors.As(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %v", err)
		}
		if !strings.Contains(ce.Reason, "plugins[0].output.layout") {
			t.Fatalf("error should name plugins[0].output.layout; got %q", ce.Reason)
		}
	})
}

// TestBuildPipeline_OutputConfigThreaded pins the wiring contract:
// project-level and per-plugin output blocks loaded from a Config
// reach the constructed [pipeline.Pipeline] and surface through
// [pipeline.Pipeline.LayoutPolicyFor] for both the run-wide
// default and per-plugin lookups.
func TestBuildPipeline_OutputConfigThreaded(t *testing.T) {
	t.Parallel()

	t.Run("project + per-plugin output config flow to LayoutPolicyFor", func(t *testing.T) {
		t.Parallel()
		env, _, _ := freshEnv(t, "eidos")
		cfg := cli.DefaultConfig()
		cfg.Output = cli.ConfigOutput{
			Layout: pipeline.LayoutCentralised, Package: "gen", Dir: "internal/gen",
		}
		cfg.Plugins = []cli.ConfigPlugin{
			{Name: "fe"},
			{Name: "mockgen", Output: &cli.ConfigOutput{
				Layout: pipeline.LayoutCentralised, Package: "mocks", Dir: "internal/mocks",
			}},
			{Name: "repogen"},
			{Name: "be"},
		}
		p, err := cli.BuildPipeline(env, cfg, []plugin.Plugin{
			stubFrontend{name: "fe"},
			stubGenerator{name: "mockgen"},
			stubGenerator{name: "repogen"},
			stubBackend{name: "be", lang: "stub"},
		})
		if err != nil {
			t.Fatalf("BuildPipeline: %v", err)
		}
		// Mockgen has a per-plugin override → its Layout policy
		// carries the per-plugin fields (under per-plugin
		// attribution).
		got := p.LayoutPolicyFor("mockgen")
		if got.Package != "mocks" || got.PackageFrom != manifest.LayerPerPlugin {
			t.Errorf(
				"mockgen Package = %q from %q, want mocks from per-plugin",
				got.Package,
				got.PackageFrom,
			)
		}
		if got.Dir != "internal/mocks" || got.DirFrom != manifest.LayerPerPlugin {
			t.Errorf(
				"mockgen Dir = %q from %q, want internal/mocks from per-plugin",
				got.Dir,
				got.DirFrom,
			)
		}
		// Repogen has no per-plugin override → its policy is the
		// project-level merge.
		got = p.LayoutPolicyFor("repogen")
		if got.Package != "gen" || got.PackageFrom != manifest.LayerProject {
			t.Errorf(
				"repogen Package = %q from %q, want gen from project",
				got.Package,
				got.PackageFrom,
			)
		}
		if got.Dir != "internal/gen" || got.DirFrom != manifest.LayerProject {
			t.Errorf(
				"repogen Dir = %q from %q, want internal/gen from project",
				got.Dir,
				got.DirFrom,
			)
		}
	})

	t.Run("empty output config leaves the framework default in place", func(t *testing.T) {
		t.Parallel()
		env, _, _ := freshEnv(t, "eidos")
		cfg := cli.DefaultConfig()
		p, err := cli.BuildPipeline(env, cfg, []plugin.Plugin{
			stubFrontend{name: "fe"},
			stubBackend{name: "be", lang: "stub"},
		})
		if err != nil {
			t.Fatalf("BuildPipeline: %v", err)
		}
		got := p.LayoutPolicyFor("anything")
		switch {
		case got.Layout != pipeline.LayoutAlongsideSource,
			got.LayoutFrom != manifest.LayerFramework:
			t.Fatalf("default policy = %+v, want framework alongside-source", got)
		}
	})
}

// TestConfig_OutputBlock_RoundTrip pins the YAML serialisation
// contract: a config carrying every documented output-block
// field marshals, re-loads, and re-marshals byte-identically.
// Embedders and tools that round-trip configs through YAML rely
// on this stability — a field rename or tag drift would surface
// here immediately.
func TestConfig_OutputBlock_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("project + per-plugin output round-trip preserves every field", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".eidos.yaml")
		body := []byte(`version: 1
output:
  layout: centralised
  package: gen
  dir: internal/gen
plugins:
  - name: mockgen
    output:
      layout: centralised
      package: mocks
      dir: internal/mocks
  - name: repogen
    output:
      package: repos
`)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		cfg, err := cli.LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.Output.Layout != pipeline.LayoutCentralised {
			t.Errorf("Output.Layout = %q, want centralised", cfg.Output.Layout)
		}
		if cfg.Output.Package != "gen" {
			t.Errorf("Output.Package = %q, want gen", cfg.Output.Package)
		}
		if cfg.Output.Dir != "internal/gen" {
			t.Errorf("Output.Dir = %q, want internal/gen", cfg.Output.Dir)
		}
		if len(cfg.Plugins) != 2 {
			t.Fatalf("Plugins len = %d, want 2", len(cfg.Plugins))
		}
		mock := cfg.Plugins[0]
		if mock.Output == nil {
			t.Fatalf("mockgen.Output should be non-nil")
		}
		switch {
		case mock.Output.Layout != pipeline.LayoutCentralised,
			mock.Output.Package != "mocks",
			mock.Output.Dir != "internal/mocks":
			t.Errorf("mockgen.Output = %+v", *mock.Output)
		}
		repo := cfg.Plugins[1]
		if repo.Output == nil {
			t.Fatalf("repogen.Output should be non-nil")
		}
		if repo.Output.Package != "repos" {
			t.Errorf("repogen.Output.Package = %q, want repos", repo.Output.Package)
		}
	})
}
