// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.thesmos.sh/eidos/core/diag"
)

// Env is the process-level context every command consumes. It
// bundles the IO surfaces (stdin/stdout/stderr/diag sink), the
// working directory, and the consumer's hardcoded brand identifier.
//
// Brand is the load-bearing field. The consumer's binary sets it
// once at program start, never via CLI flag or config file. It
// flows into [plugin.BackendContext.Brand] for the rendered file
// header/footer markers, and into default paths for project-local
// state under `.<Brand>/`.
type Env struct {
	// Brand identifies the tool. The reference binary uses "eidos";
	// downstream rebrandings use their own. The empty string is
	// invalid — every command's Execute path rejects it as a
	// configuration error before any pipeline work happens.
	Brand string

	// Workdir is the directory all relative paths resolve against.
	// Typically the process's current working directory.
	Workdir string

	// Stdin / Stdout / Stderr are the IO surfaces commands write to.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// Diag is the diagnostic sink commands emit Info / Warn / Error
	// records into. The cli package's diagnostic formatters render
	// the recorded entries to Stdout / Stderr in the configured
	// format (text / json).
	Diag *diag.Sink
}

// NewEnv returns an [Env] populated with sensible defaults — cwd as
// Workdir, os.Stdin/Stdout/Stderr, a fresh [diag.Sink]. brand is
// the consumer's hardcoded brand identity (typically the binary's
// name). Returns an error when brand is empty; the command surface
// can't operate without a brand to anchor state paths to.
func NewEnv(brand string) (*Env, error) {
	if brand == "" {
		return nil, &ConfigError{Reason: "Env.Brand is required (the consumer's hardcoded tool identity)"}
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("cli: working directory unavailable: %w", err)
	}
	return &Env{
		Brand:   brand,
		Workdir: wd,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Diag:    diag.New(),
	}, nil
}

// StateDir returns the default project-local state directory:
// `<Workdir>/.<Brand>`. Used as the parent of manifest.json and the
// cache subdirectory unless the config file overrides individual
// paths.
func (e *Env) StateDir() string { return filepath.Join(e.Workdir, "."+e.Brand) }

// ManifestPath returns the default manifest path:
// `<Workdir>/.<Brand>/manifest.json`. Callers consulting the config
// file should prefer the config's `manifest.path` field when it
// is non-empty.
func (e *Env) ManifestPath() string { return filepath.Join(e.StateDir(), "manifest.json") }

// CacheDir returns the default cache directory:
// `<Workdir>/.<Brand>/cache`. Callers consulting the config file
// should prefer the config's `cache.dir` field when it is
// non-empty.
func (e *Env) CacheDir() string { return filepath.Join(e.StateDir(), "cache") }

// ConfigFileName returns the default config-file basename:
// `.<Brand>.yaml`. The config-file discovery routine walks up from
// Workdir looking for this name.
func (e *Env) ConfigFileName() string { return "." + e.Brand + ".yaml" }
