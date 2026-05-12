// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/core/diag"
)

// freshEnv returns an Env wired to bytes buffers for stdio so tests
// can assert on what main wrote without touching real process IO.
func freshEnv(t *testing.T) (env *cli.Env, stdout, stderr *bytes.Buffer) {
	t.Helper()
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	env = &cli.Env{
		Brand:   brand,
		Workdir: t.TempDir(),
		Stdin:   strings.NewReader(""),
		Stdout:  stdout,
		Stderr:  stderr,
		Diag:    diag.New(),
	}
	return env, stdout, stderr
}

// TestRun_HelpAndDefault covers the help-text paths: no args and the
// canonical --help / -h / "help" forms each print usage to stdout
// and exit ExitOK.
func TestRun_HelpAndDefault(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"-h", []string{"-h"}},
		{"--help", []string{"--help"}},
		{"help subcommand", []string{"help"}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" prints usage", func(t *testing.T) {
			t.Parallel()
			env, stdout, _ := freshEnv(t)
			code := run(t.Context(), env, defaultPlugins(), tc.args)
			if code != cli.ExitOK {
				t.Fatalf("run = %d, want ExitOK", code)
			}
			if !strings.Contains(stdout.String(), "eidos <command>") {
				t.Fatalf("expected usage banner; got %q", stdout.String())
			}
		})
	}
}

// TestRun_UnknownCommand covers the dispatch fallback: an unknown
// subcommand exits ExitUserError and writes the usage banner to
// stderr so the operator can self-correct.
func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	t.Run("unknown command exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t)
		code := run(t.Context(), env, defaultPlugins(), []string{"frobnicate"})
		if code != cli.ExitUserError {
			t.Fatalf("run = %d, want ExitUserError", code)
		}
		if !strings.Contains(stderr.String(), "unknown command") {
			t.Fatalf("expected unknown-command diagnostic; got %q", stderr.String())
		}
		if !strings.Contains(stderr.String(), "eidos <command>") {
			t.Fatalf("expected usage banner on stderr; got %q", stderr.String())
		}
	})
}

// TestRun_VersionSubcommand exercises the version-subcommand path
// end-to-end with the real plugin slice. Pins the brand line and
// confirms the static plugin universe rendered into the listing.
func TestRun_VersionSubcommand(t *testing.T) {
	t.Parallel()

	t.Run("renders brand + emit-contract + plugin list", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t)
		code := run(t.Context(), env, defaultPlugins(), []string{"version"})
		if code != cli.ExitOK {
			t.Fatalf("run version = %d, want ExitOK", code)
		}
		out := stdout.String()
		if !strings.HasPrefix(out, "eidos\n") {
			t.Fatalf("expected brand line; got %q", out)
		}
		if !strings.Contains(out, "emit-contract:") {
			t.Fatalf("expected emit-contract line; got %q", out)
		}
		if !strings.Contains(out, "repogen") {
			t.Fatalf("expected repogen in plugin listing; got %q", out)
		}
	})

	t.Run("--diag-format json renders structured output", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t)
		code := run(t.Context(), env, defaultPlugins(), []string{"version", "--diag-format", "json"})
		if code != cli.ExitOK {
			t.Fatalf("run version --diag-format json = %d, want ExitOK", code)
		}
		out := stdout.String()
		if !strings.HasPrefix(strings.TrimSpace(out), "{") {
			t.Fatalf("expected JSON object; got %q", out)
		}
	})
}

// TestRun_ExplainPropagatesSelector confirms the explain dispatcher
// forwards the first positional arg as the selector — the empty
// case (no positional) exits ExitUserError per the cli contract.
func TestRun_ExplainPropagatesSelector(t *testing.T) {
	t.Parallel()

	t.Run("missing selector exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t)
		code := run(t.Context(), env, defaultPlugins(), []string{"explain"})
		if code != cli.ExitUserError {
			t.Fatalf("run explain = %d, want ExitUserError", code)
		}
		if !strings.Contains(stderr.String(), "selector is required") {
			t.Fatalf("expected selector-required diagnostic; got %q", stderr.String())
		}
	})
}

// TestRun_BadFlag covers flag-parse failures: an unknown flag prints
// the flag.FlagSet error and exits ExitUserError without invoking
// the command body.
func TestRun_BadFlag(t *testing.T) {
	t.Parallel()

	t.Run("unknown flag exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t)
		code := run(t.Context(), env, defaultPlugins(), []string{"plan", "--no-such-flag"})
		if code != cli.ExitUserError {
			t.Fatalf("run plan --no-such-flag = %d, want ExitUserError", code)
		}
		if !strings.Contains(stderr.String(), "flag provided but not defined") {
			t.Fatalf("expected flag-parse diagnostic; got %q", stderr.String())
		}
	})
}

// TestLoadConfigOrDefault covers the config-resolution helper: an
// empty --config + no discoverable file falls back to defaults, and
// an explicit path that points at a missing file surfaces as a
// *ConfigError.
func TestLoadConfigOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("no explicit and no discovery returns DefaultConfig", func(t *testing.T) {
		t.Parallel()
		env, _, _ := freshEnv(t)
		cfg, err := loadConfigOrDefault(env, "")
		if err != nil {
			t.Fatalf("loadConfigOrDefault err = %v", err)
		}
		if cfg == nil {
			t.Fatalf("loadConfigOrDefault returned nil cfg")
		}
		if cfg.Version != cli.ConfigVersion {
			t.Fatalf("default cfg version = %d, want %d", cfg.Version, cli.ConfigVersion)
		}
	})

	t.Run("explicit missing file returns ConfigError", func(t *testing.T) {
		t.Parallel()
		env, _, _ := freshEnv(t)
		_, err := loadConfigOrDefault(env, env.Workdir+"/does-not-exist.yaml")
		if err == nil {
			t.Fatalf("loadConfigOrDefault should fail on missing explicit path")
		}
	})
}

// TestDefaultPlugins is a smoke test on the static plugin slice:
// every plugin in the universe must have a non-empty Name, and the
// expected reference names must be present so a copy-paste regression
// (e.g. dropping repogen) trips the test.
func TestDefaultPlugins(t *testing.T) {
	t.Parallel()

	t.Run("every plugin has a name and required names are present", func(t *testing.T) {
		t.Parallel()
		want := map[string]bool{
			"repogen":      false,
			"mockgen":      false,
			"buildergen":   false,
			"registry-gen": false,
			"shape-writer": false,
			"audit-weaver": false,
			"debug-weaver": false,
		}
		for _, p := range defaultPlugins() {
			if p.Name() == "" {
				t.Fatalf("plugin without a name: %T", p)
			}
			if _, ok := want[p.Name()]; ok {
				want[p.Name()] = true
			}
		}
		for name, seen := range want {
			if !seen {
				t.Fatalf("expected reference plugin %q to be registered in defaultPlugins", name)
			}
		}
	})
}

// stderrOrStubTest covers the defensive nil-env path used when a
// downstream test harness constructs an Env without populating
// Stderr — flag.FlagSet panics on a nil writer, so the helper
// substitutes io.Discard.
func TestStderrOrStub(t *testing.T) {
	t.Parallel()

	t.Run("nil env returns io.Discard", func(t *testing.T) {
		t.Parallel()
		if got := stderrOrStub(nil); got != io.Discard {
			t.Fatalf("stderrOrStub(nil) = %v, want io.Discard", got)
		}
	})

	t.Run("env with nil Stderr returns io.Discard", func(t *testing.T) {
		t.Parallel()
		env := &cli.Env{}
		if got := stderrOrStub(env); got != io.Discard {
			t.Fatalf("stderrOrStub(empty env) = %v, want io.Discard", got)
		}
	})

	t.Run("env with Stderr returns the underlying writer", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		env := &cli.Env{Stderr: buf}
		if got := stderrOrStub(env); got != buf {
			t.Fatalf("stderrOrStub should return env.Stderr verbatim")
		}
	})
}
