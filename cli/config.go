// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.thesmos.sh/eidos/core/directive"
)

// ConfigVersion is the schema version the loader accepts. Bump
// whenever the on-disk format changes incompatibly.
const ConfigVersion = 1

// Config is the in-memory representation of the .<brand>.yaml
// config file. The loader applies defaults during parse, so every
// field is populated on a successful Load — fields the file
// omitted carry the documented default. Callers consuming a
// *Config never see nil sub-records on present fields.
type Config struct {
	// Version is the schema version of the on-disk file. Must
	// equal [ConfigVersion]; mismatch surfaces as a [*ConfigError].
	Version int `yaml:"version"`

	// Sources lists the per-frontend input descriptors.
	Sources []ConfigSource `yaml:"sources,omitempty"`

	// Plugins selects and configures plugins from the consumer's
	// statically-imported plugin universe. A plugin named here
	// that isn't in the consumer's slice is a user error.
	Plugins []ConfigPlugin `yaml:"plugins,omitempty"`

	// Sink configures the output sink. Defaults to a disk sink
	// rooted at the working directory.
	Sink ConfigSink `yaml:"sink,omitempty"`

	// Cache configures the build cache. Defaults to enabled with
	// the brand-derived directory.
	Cache ConfigCache `yaml:"cache,omitempty"`

	// Manifest configures the manifest output. Defaults to the
	// brand-derived path.
	Manifest ConfigManifest `yaml:"manifest,omitempty"`

	// Directives configures the directive parser.
	Directives ConfigDirectives `yaml:"directives,omitempty"`

	// Parallel names the phases the pipeline runs in parallel mode.
	Parallel []string `yaml:"parallel,omitempty"`

	// Envelope configures the header / footer envelope.
	Envelope ConfigEnvelope `yaml:"envelope,omitempty"`

	// Verbose mirrors the --verbose flag; the CLI flag overrides
	// when set.
	Verbose bool `yaml:"verbose,omitempty"`
}

// ConfigSource is one frontend + input-pattern pair.
type ConfigSource struct {
	// Frontend names the frontend plugin (matches Plugin.Name()).
	Frontend string `yaml:"frontend"`

	// Patterns are the input descriptors passed to Frontend.Load.
	// Typically Go-style import paths (`./...`) or globs.
	Patterns []string `yaml:"patterns,omitempty"`
}

// ConfigPlugin selects and configures one plugin.
type ConfigPlugin struct {
	// Name matches Plugin.Name() on a plugin in the consumer's
	// static slice.
	Name string `yaml:"name"`

	// Enabled toggles the plugin on / off. Defaults to true when
	// omitted.
	Enabled *bool `yaml:"enabled,omitempty"`

	// Options is the plugin's typed options map. The pipeline
	// validates each entry against the plugin's OptionsSchema.
	Options map[string]any `yaml:"options,omitempty"`
}

// IsEnabled reports whether the plugin is active. Defaults to true
// when the file omits the Enabled field.
func (p ConfigPlugin) IsEnabled() bool { return p.Enabled == nil || *p.Enabled }

// ConfigSink configures the output sink.
type ConfigSink struct {
	// Kind selects the sink implementation. One of: "disk",
	// "memory", "multi", "stdout". Defaults to "disk".
	Kind string `yaml:"kind,omitempty"`

	// Options is a kind-specific configuration map. Currently
	// unused for "disk" (routing handles layout); reserved for
	// future kinds.
	Options map[string]any `yaml:"options,omitempty"`
}

// ConfigCache configures the build cache.
type ConfigCache struct {
	// Enabled toggles cache usage. Defaults to true when omitted.
	Enabled *bool `yaml:"enabled,omitempty"`

	// Dir overrides the cache directory. Empty falls back to
	// `<Env.Workdir>/.<Brand>/cache`.
	Dir string `yaml:"dir,omitempty"`
}

// IsEnabled reports whether the cache is active. Defaults to true.
func (c ConfigCache) IsEnabled() bool { return c.Enabled == nil || *c.Enabled }

// ConfigManifest configures the manifest output.
type ConfigManifest struct {
	// Path overrides the manifest path. Empty falls back to
	// `<Env.Workdir>/.<Brand>/manifest.json`.
	Path string `yaml:"path,omitempty"`
}

// ConfigDirectives configures the directive parser.
type ConfigDirectives struct {
	// Prefix is the directive name prefix the parser strips
	// before schema dispatch (e.g. "gen" for "+gen:repo").
	// Defaults to "gen" when omitted.
	Prefix string `yaml:"prefix,omitempty"`
}

// ConfigEnvelope configures the header / footer envelope.
type ConfigEnvelope struct {
	// HeaderPrefix adds lines before the standard header marker.
	HeaderPrefix []string `yaml:"header_prefix,omitempty"`

	// HeaderSuffix adds lines after the standard header.
	HeaderSuffix []string `yaml:"header_suffix,omitempty"`

	// FooterSuffix adds lines after the standard footer.
	FooterSuffix []string `yaml:"footer_suffix,omitempty"`

	// SourcesOverride replaces the auto-aggregated Source: header
	// lines. Useful for programmatic invocations that have no
	// source files.
	SourcesOverride []string `yaml:"sources_override,omitempty"`
}

// DefaultConfig returns a *Config populated with the documented
// defaults. Used as the seed value the YAML decoder mutates,
// and returned directly when no config file exists.
func DefaultConfig() *Config {
	return &Config{
		Version: ConfigVersion,
		Sources: []ConfigSource{},
		Sink: ConfigSink{
			Kind: SinkKindDisk,
		},
		Cache:    ConfigCache{},
		Manifest: ConfigManifest{},
		Directives: ConfigDirectives{
			Prefix: directive.DefaultPrefix,
		},
	}
}

// Sink kind identifiers accepted by [ConfigSink.Kind].
const (
	SinkKindDisk   = "disk"
	SinkKindMemory = "memory"
	SinkKindMulti  = "multi"
	SinkKindStdout = "stdout"
)

// LoadConfig reads, parses, and validates a config file at path.
// Returns the populated *Config or a *ConfigError carrying the
// file:line:col of the first diagnostic. An empty path returns
// the default config without touching disk.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &ConfigError{Path: path, Reason: "config file not found"}
		}
		return nil, &ConfigError{Path: path, Reason: "read failed: " + err.Error()}
	}
	c, perr := parseConfig(path, raw)
	if perr != nil {
		return nil, perr
	}
	if err := validateConfig(c, path); err != nil {
		return nil, err
	}
	return c, nil
}

// DiscoverConfig walks up from start looking for the file named
// filename. Returns the resolved path and true when found, the
// empty string and false otherwise. Used by callers that want to
// resolve a default config-file path against [Env.Workdir].
//
// The walk stops at the filesystem root.
func DiscoverConfig(start, filename string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, filename)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// validateConfig enforces structural invariants the YAML decoder
// can't express: known Version, plugins must have names, sources
// must have a frontend, etc.
func validateConfig(c *Config, path string) error {
	if c.Version == 0 {
		c.Version = ConfigVersion
	}
	if c.Version != ConfigVersion {
		return &ConfigError{
			Path:   path,
			Reason: fmt.Sprintf("unsupported config version %d (expected %d)", c.Version, ConfigVersion),
		}
	}
	if c.Directives.Prefix == "" {
		c.Directives.Prefix = directive.DefaultPrefix
	}
	if c.Sink.Kind == "" {
		c.Sink.Kind = SinkKindDisk
	}
	for i, src := range c.Sources {
		if src.Frontend == "" {
			return &ConfigError{Path: path, Reason: fmt.Sprintf("sources[%d]: frontend is required", i)}
		}
	}
	for i, p := range c.Plugins {
		if p.Name == "" {
			return &ConfigError{Path: path, Reason: fmt.Sprintf("plugins[%d]: name is required", i)}
		}
	}
	switch c.Sink.Kind {
	case SinkKindDisk, SinkKindMemory, SinkKindMulti, SinkKindStdout:
		// known kinds.
	default:
		return &ConfigError{
			Path: path,
			Reason: fmt.Sprintf(
				"sink.kind %q is not recognised (one of: %s, %s, %s, %s)",
				c.Sink.Kind, SinkKindDisk, SinkKindMemory, SinkKindMulti, SinkKindStdout,
			),
		}
	}
	return nil
}
