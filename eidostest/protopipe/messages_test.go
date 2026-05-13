// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protopipe_test

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/eidostest/protopipe"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/frontend/protobuf"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/buildergen"
)

// TestRun_MessagesBuildergen_BuilderShape adds a loose assertion
// alongside the presence check: UserBuilder carries a Build
// method (the universal builder closer). Per-field With<Name>
// setters require the protogo bridge to expose Go-exported
// names; without the bridge the source-side proto field names
// stay lowercase and buildergen's exported-field filter drops
// them, so the pin here covers only the Build method shape.
func TestRun_MessagesBuildergen_BuilderShape(t *testing.T) {
	t.Parallel()

	t.Run("UserBuilder carries a Build method", func(t *testing.T) {
		t.Parallel()
		root := messagesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir:  root,
			Generators: []plugin.Generator{buildergen.New()},
			Backend:    backend_golang.New(),
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		methods := result.Store.Emit().Methods().Items()
		var foundBuild bool
		for _, m := range methods {
			owner, ok := m.Owner.(*emit.Struct)
			if !ok || owner.Name != "UserBuilder" {
				continue
			}
			if m.Name == "Build" {
				foundBuild = true
				break
			}
		}
		if !foundBuild {
			t.Fatalf("UserBuilder.Build method missing from emit graph")
		}
	})
}

// TestRun_MessagesBuildergen pins the directive-driven buildergen
// gate against a proto-derived message: User carries +gen:builder
// in the messages fixture; buildergen produces a UserBuilder emit
// struct with setters covering its non-oneof fields. The exact
// shape of any oneof-variant setter is a downstream-plugin
// choice that this test does not over-pin.
func TestRun_MessagesBuildergen(t *testing.T) {
	t.Parallel()

	t.Run("proto message carrying +gen:builder produces a Builder struct", func(t *testing.T) {
		t.Parallel()
		root := messagesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir:  root,
			Generators: []plugin.Generator{buildergen.New()},
			Backend:    backend_golang.New(),
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		structs := result.Store.Emit().Structs().Items()
		var found *emit.Struct
		for _, s := range structs {
			if s.Name == "UserBuilder" {
				found = s
				break
			}
		}
		if found == nil {
			names := make([]string, 0, len(structs))
			for _, s := range structs {
				names = append(names, s.Name)
			}
			t.Fatalf("expected an emit struct named UserBuilder; got %+v", names)
		}
	})
}

// TestRun_MessagesExplainOneofWalk drives the real protobuf
// frontend through cli.ExplainCommand and asserts the production
// proto.oneof.message / proto.oneof.interface meta keys land in
// the rendered output. This closes the end-to-end gap a
// cli-package-internal test left open — the production meta-key
// names rather than test-only stand-ins.
func TestRun_MessagesExplainOneofWalk(t *testing.T) {
	t.Parallel()

	t.Run("explain on the synthesized interface renders proto.oneof.message", func(t *testing.T) {
		t.Parallel()
		stdout, code := runExplainAgainstMessages(t, "messages.Container_choice")
		if code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK; stdout:\n%s", code, stdout)
		}
		want := "proto.oneof.message = eidos.protobuf.testdata.messages.Container"
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in explain output; got:\n%s", want, stdout)
		}
	})

	t.Run("explain on a variant field renders proto.oneof.interface", func(t *testing.T) {
		t.Parallel()
		stdout, code := runExplainAgainstMessages(t, "messages.Container.text")
		if code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK; stdout:\n%s", code, stdout)
		}
		want := "proto.oneof.interface = eidos.protobuf.testdata.messages.Container_choice"
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in explain output; got:\n%s", want, stdout)
		}
	})
}

// runExplainAgainstMessages builds an [cli.ExplainCommand]
// against the messages fixture wired with the real protobuf
// frontend + Go backend and returns stdout plus the exit code.
// The fixture's `dir` option threads through the cli's config
// surface so the frontend's Load runs against the right tree.
func runExplainAgainstMessages(t *testing.T, selector string) (string, int) {
	t.Helper()
	root := messagesFixtureRoot(t)
	cfg := cli.DefaultConfig()
	cfg.Plugins = []cli.ConfigPlugin{{
		Name: protobuf.FrontendName,
		Options: map[string]any{
			"dir": root,
		},
	}}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := &cli.Env{
		Brand:   "eidos",
		Workdir: t.TempDir(),
		Stdout:  stdout,
		Stderr:  stderr,
	}
	cmd := &cli.ExplainCommand{Config: cli.ExplainConfig{
		File: cfg,
		Plugins: []plugin.Plugin{
			protobuf.New(),
			backend_golang.New(),
		},
		Selector: selector,
	}}
	code := cmd.Execute(t.Context(), env)
	if code != cli.ExitOK {
		t.Logf("explain stderr:\n%s", stderr.String())
	}
	return stdout.String(), code
}

// messagesFixtureRoot resolves the absolute path of the
// frontend/protobuf/testdata/messages directory.
func messagesFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	return filepath.Join(repoRoot, "frontend", "protobuf", "testdata", "messages")
}
