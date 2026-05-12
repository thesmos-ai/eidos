// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
)

// TestCheckCommand_NoDrift covers the happy path: when the
// on-disk file matches the pipeline's output, the command
// reports zero drift and exits ExitOK.
func TestCheckCommand_NoDrift(t *testing.T) {
	t.Parallel()

	t.Run("matching on-disk content reports no drift", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		body := []byte("// generated\npackage x\n")
		tgt := emit.Target{Dir: "out", Filename: "hello.go", Package: "x"}
		// Seed the on-disk state to match what the pipeline will write.
		if err := os.MkdirAll(filepath.Join(env.Workdir, tgt.Dir), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(env.Workdir, tgt.Dir, tgt.Filename), body, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		cmd := &cli.CheckCommand{Config: cli.CheckConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				writingBackend{name: "be", body: body, tgt: tgt},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
		if !strings.Contains(stdout.String(), "no drift detected") {
			t.Fatalf("expected no-drift line; got %q", stdout.String())
		}
	})
}

// TestCheckCommand_ContentDrift covers the drift path: when the
// on-disk file's bytes differ from the pipeline's output, the
// command exits ExitCheckDrift.
func TestCheckCommand_ContentDrift(t *testing.T) {
	t.Parallel()

	t.Run("different on-disk bytes report drift", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		body := []byte("// generated\npackage x\n")
		stale := []byte("// stale\npackage x\n")
		tgt := emit.Target{Dir: "out", Filename: "hello.go", Package: "x"}
		if err := os.MkdirAll(filepath.Join(env.Workdir, tgt.Dir), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(env.Workdir, tgt.Dir, tgt.Filename), stale, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		cmd := &cli.CheckCommand{Config: cli.CheckConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				writingBackend{name: "be", body: body, tgt: tgt},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitCheckDrift {
			t.Fatalf("Execute = %d, want ExitCheckDrift", code)
		}
		if !strings.Contains(stdout.String(), "content differs") {
			t.Fatalf("expected content-differs line; got %q", stdout.String())
		}
	})
}

// TestCheckCommand_MissingFile covers the missing-on-disk case:
// the pipeline produced a file that doesn't exist on disk — also
// drift, treated as such by CI gates.
func TestCheckCommand_MissingFile(t *testing.T) {
	t.Parallel()

	t.Run("file missing from disk is drift", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		body := []byte("// generated\npackage x\n")
		tgt := emit.Target{Dir: "out", Filename: "hello.go", Package: "x"}
		// Don't seed the disk — the pipeline output has no counterpart.
		cmd := &cli.CheckCommand{Config: cli.CheckConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				writingBackend{name: "be", body: body, tgt: tgt},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitCheckDrift {
			t.Fatalf("Execute = %d, want ExitCheckDrift", code)
		}
		if !strings.Contains(stdout.String(), "missing from disk") {
			t.Fatalf("expected missing-from-disk line; got %q", stdout.String())
		}
	})
}

// TestCheckCommand_ConfigError covers the user-error path.
func TestCheckCommand_ConfigError(t *testing.T) {
	t.Parallel()

	t.Run("missing plugin exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t, "eidos")
		cfg := cli.DefaultConfig()
		cfg.Plugins = []cli.ConfigPlugin{{Name: "ghost"}}
		cmd := &cli.CheckCommand{Config: cli.CheckConfig{
			File: cfg,
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				stubBackend{name: "be", lang: "stub"},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitUserError {
			t.Fatalf("Execute = %d, want ExitUserError", code)
		}
		if !strings.Contains(stderr.String(), "ghost") {
			t.Fatalf("stderr should name the missing plugin; got %q", stderr.String())
		}
	})
}

// TestCheckCommand_EmptyPipelineNoDrift covers the no-output case:
// a pipeline with a backend that writes nothing reports zero
// outputs checked and zero drift.
func TestCheckCommand_EmptyPipelineNoDrift(t *testing.T) {
	t.Parallel()

	t.Run("pipeline with no sink writes reports no drift", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		cmd := &cli.CheckCommand{Config: cli.CheckConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				stubBackend{name: "be", lang: "stub"},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
		if !strings.Contains(stdout.String(), "0 output") {
			t.Fatalf("expected zero-output summary; got %q", stdout.String())
		}
	})
}
