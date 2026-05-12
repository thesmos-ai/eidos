// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
)

// stubFrontend / stubBackend are the minimal plugin role
// implementations Plan / Prune tests need to satisfy
// pipeline.Build's "at least one frontend, exactly one backend"
// invariant.
type stubFrontend struct{ name string }

func (f stubFrontend) Name() string                       { return f.name }
func (stubFrontend) Load(_ *plugin.FrontendContext) error { return nil }

type stubBackend struct{ name, lang string }

func (b stubBackend) Name() string                        { return b.name }
func (b stubBackend) Language() string                    { return b.lang }
func (stubBackend) Render(_ *plugin.BackendContext) error { return nil }

// stubGenerator is the minimal [plugin.Generator] role for tests
// that need a named generator to exist in the plugin universe
// (for per-plugin output-config wiring) without contributing any
// emit graph.
type stubGenerator struct{ name string }

func (g stubGenerator) Name() string                            { return g.name }
func (stubGenerator) Generate(_ *plugin.GeneratorContext) error { return nil }

// TestPlanCommand_Text covers the human-readable plan output.
// Build invariants force a frontend + backend pair so the
// rendering walks every plan section.
func TestPlanCommand_Text(t *testing.T) {
	t.Parallel()

	t.Run("renders frontend / annotator / generator / backend sections", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		cmd := &cli.PlanCommand{Config: cli.PlanConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				stubBackend{name: "be", lang: "stub"},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
		out := stdout.String()
		for _, label := range []string{"frontends:", "annotators:", "generators:", "backend:"} {
			if !strings.Contains(out, label) {
				t.Fatalf("expected %q in output; got %q", label, out)
			}
		}
		if !strings.Contains(out, "- fe") {
			t.Fatalf("expected frontend name; got %q", out)
		}
		if !strings.Contains(out, "backend: be") {
			t.Fatalf("expected backend line; got %q", out)
		}
	})
}

// TestPlanCommand_JSON covers the NDJSON plan rendering.
func TestPlanCommand_JSON(t *testing.T) {
	t.Parallel()

	t.Run("renders a single JSON object with section keys", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		cmd := &cli.PlanCommand{Config: cli.PlanConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				stubBackend{name: "be", lang: "stub"},
			},
			Format: cli.DiagFormatJSON,
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
		var out struct {
			Frontends  []string `json:"frontends"`
			Annotators []struct {
				Priority int      `json:"priority"`
				Plugins  []string `json:"plugins"`
			} `json:"annotators"`
			Generators []struct {
				Priority int      `json:"priority"`
				Plugins  []string `json:"plugins"`
			} `json:"generators"`
			Backend string `json:"backend"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
			t.Fatalf("not valid JSON: %v\n%q", err, stdout.String())
		}
		if len(out.Frontends) != 1 || out.Frontends[0] != "fe" {
			t.Fatalf("Frontends = %+v, want [fe]", out.Frontends)
		}
		if out.Backend != "be" {
			t.Fatalf("Backend = %q, want be", out.Backend)
		}
	})
}

// TestPlanCommand_ConfigError covers the build-failure path: a
// config naming a missing plugin exits ExitUserError.
func TestPlanCommand_ConfigError(t *testing.T) {
	t.Parallel()

	t.Run("plugin named in config but not in slice exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t, "eidos")
		cfg := cli.DefaultConfig()
		cfg.Plugins = []cli.ConfigPlugin{{Name: "ghost"}}
		cmd := &cli.PlanCommand{Config: cli.PlanConfig{
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

// TestPlanCommand_StdinNilToleratedWhenUnused covers a regression
// guard: a *bytes.Buffer doesn't have an io.Reader fallback when
// the field is unused; this confirms PlanCommand never touches
// env.Stdin.
func TestPlanCommand_StdinNilToleratedWhenUnused(t *testing.T) {
	t.Parallel()

	t.Run("nil env.Stdin is acceptable", func(t *testing.T) {
		t.Parallel()
		env := &cli.Env{
			Brand:   "eidos",
			Workdir: t.TempDir(),
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
		}
		cmd := &cli.PlanCommand{Config: cli.PlanConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				stubBackend{name: "be", lang: "stub"},
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
	})
}

// TestPlanCommand_RegisterFlags_Routing pins the routing-flag
// wiring on PlanCommand — parsed values land on
// [cli.PlanConfig.Routing].
func TestPlanCommand_RegisterFlags_Routing(t *testing.T) {
	t.Parallel()

	t.Run("Routing flags parse onto PlanCommand.Config.Routing", func(t *testing.T) {
		t.Parallel()
		var cmd cli.PlanCommand
		fs := flag.NewFlagSet("plan", flag.ContinueOnError)
		cmd.RegisterFlags(fs)
		assertRoutingFlagsParse(t, fs, &cmd.Config.Routing)
	})
}

// TestPlanCommand_ResolvedPolicySection pins the routing-policy
// section in plan output: every registered plugin appears under
// a "resolved policy" heading with its merged Layout / Package /
// Dir values and the precedence-layer attribution for each.
// Two consecutive invocations against identical inputs must
// produce byte-identical text.
func TestPlanCommand_ResolvedPolicySection(t *testing.T) {
	t.Parallel()

	t.Run("text output includes per-plugin resolved-policy section", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		cmd := &cli.PlanCommand{Config: cli.PlanConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				stubGenerator{name: "rg"},
				stubBackend{name: "be", lang: "stub"},
			},
			Routing: cli.RoutingFlags{
				Layout:  pipeline.LayoutCentralised,
				Package: "gen",
			},
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
		out := stdout.String()
		for _, want := range []string{
			"resolved policy",
			"rg:",
			"layout=centralised",
			"package=gen",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("plan output missing %q; got:\n%s", want, out)
			}
		}
	})

	t.Run("byte-stable across two invocations with identical inputs", func(t *testing.T) {
		t.Parallel()
		env1, stdout1, _ := freshEnv(t, "eidos")
		env2, stdout2, _ := freshEnv(t, "eidos")
		mk := func() *cli.PlanCommand {
			return &cli.PlanCommand{Config: cli.PlanConfig{
				Plugins: []plugin.Plugin{
					stubFrontend{name: "fe"},
					stubGenerator{name: "rg"},
					stubBackend{name: "be", lang: "stub"},
				},
				Routing: cli.RoutingFlags{
					Layout:  pipeline.LayoutCentralised,
					Package: "gen",
				},
			}}
		}
		_ = mk().Execute(t.Context(), env1)
		_ = mk().Execute(t.Context(), env2)
		if stdout1.String() != stdout2.String() {
			t.Fatalf("plan output not byte-stable across runs:\n--- run1 ---\n%s\n--- run2 ---\n%s",
				stdout1.String(), stdout2.String())
		}
	})
}
