// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
)

// writingBackend is a minimal [plugin.Backend] that writes a
// fixed body through ctx.Sink at a known target. Used to verify
// RunCommand wires the pipeline → sink path correctly.
type writingBackend struct {
	name string
	body []byte
	tgt  emit.Target
}

func (b writingBackend) Name() string   { return b.name }
func (writingBackend) Language() string { return "stub" }
func (b writingBackend) Render(ctx *plugin.BackendContext) error {
	return ctx.Sink.Write(b.tgt, b.body)
}

// failingBackend is a [plugin.Backend] that returns a non-nil
// error from Render. Used to verify RunCommand surfaces
// pipeline-level errors as ExitPipelineError.
type failingBackend struct{ name string }

func (b failingBackend) Name() string                        { return b.name }
func (failingBackend) Language() string                      { return "stub" }
func (failingBackend) Render(_ *plugin.BackendContext) error { return errors.New("backend boom") }

// TestRunCommand_WritesThroughSink covers the happy path: a
// pipeline with a stub frontend + a writing backend rooted at a
// tmpdir produces the file on disk after Run.
func TestRunCommand_WritesThroughSink(t *testing.T) {
	t.Parallel()

	t.Run("writing backend reaches disk via the configured sink", func(t *testing.T) {
		t.Parallel()
		env, _, _ := freshEnv(t, "eidos")
		body := []byte("// generated\npackage x\n")
		tgt := emit.Target{Dir: "out", Filename: "hello.go", Package: "x"}
		cmd := &cli.RunCommand{Config: cli.RunConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				writingBackend{name: "be", body: body, tgt: tgt},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
		got, err := os.ReadFile(filepath.Join(env.Workdir, tgt.Dir, tgt.Filename))
		if err != nil {
			t.Fatalf("expected sink write at target; read err=%v", err)
		}
		if !bytes.Equal(got, body) {
			t.Fatalf("file content mismatch:\n got: %q\nwant: %q", got, body)
		}
	})
}

// TestRunCommand_BackendErrorExitsPipelineError covers the
// pipeline-error path: when Render returns an error the pipeline
// records it as a Diag.Error and Run returns ErrRunHadErrors;
// the command exits ExitPipelineError.
func TestRunCommand_BackendErrorExitsPipelineError(t *testing.T) {
	t.Parallel()

	t.Run("backend Render error surfaces as ExitPipelineError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t, "eidos")
		cmd := &cli.RunCommand{Config: cli.RunConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				failingBackend{name: "be"},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitPipelineError {
			t.Fatalf("Execute = %d, want ExitPipelineError", code)
		}
		if !strings.Contains(stderr.String(), "backend boom") {
			t.Fatalf("expected backend error in diagnostics; got %q", stderr.String())
		}
	})
}

// TestRunCommand_ConfigError covers the user-error path: a config
// naming a missing plugin exits ExitUserError before any
// pipeline construction.
func TestRunCommand_ConfigError(t *testing.T) {
	t.Parallel()

	t.Run("missing plugin exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t, "eidos")
		cfg := cli.DefaultConfig()
		cfg.Plugins = []cli.ConfigPlugin{{Name: "ghost"}}
		cmd := &cli.RunCommand{Config: cli.RunConfig{
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

// TestRunCommand_NoCacheOverride covers the runtime override: a
// --no-cache flag wins over a config that has cache.enabled=true.
// We verify by running with NoCache and confirming the run still
// succeeds (the pipeline accepts cache.NewNone()).
func TestRunCommand_NoCacheOverride(t *testing.T) {
	t.Parallel()

	t.Run("NoCache flag flips the cache to None without breaking the run", func(t *testing.T) {
		t.Parallel()
		env, _, _ := freshEnv(t, "eidos")
		cfg := cli.DefaultConfig()
		enabled := true
		cfg.Cache.Enabled = &enabled
		body := []byte("// generated\npackage x\n")
		tgt := emit.Target{Dir: "out", Filename: "h.go", Package: "x"}
		cmd := &cli.RunCommand{Config: cli.RunConfig{
			File:    cfg,
			NoCache: true,
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				writingBackend{name: "be", body: body, tgt: tgt},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
	})
}
