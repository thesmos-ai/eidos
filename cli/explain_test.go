// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
)

// emittingGenerator is a [plugin.Generator] that adds a single
// fixture package to the store on Generate. Used by Explain tests
// that need a known entity in the emit graph.
type emittingGenerator struct {
	name string
	pkg  *emit.Package
}

func (g emittingGenerator) Name() string { return g.name }
func (g emittingGenerator) Generate(ctx *plugin.GeneratorContext) error {
	return ctx.Store.Emit().AddPackage(g.pkg)
}

// TestExplainCommand_EntitySelector covers the entity-only form:
// `pkg.QName` resolves to the matching emit entity and prints its
// kind summary.
func TestExplainCommand_EntitySelector(t *testing.T) {
	t.Parallel()

	t.Run("resolved entity prints Kind line", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		gen := emittingGenerator{
			name: "fixturegen",
			pkg: &emit.Package{
				Name: "users", Path: "users",
				Structs: []*emit.Struct{{
					Name: "User", Package: "users",
					Target: emit.Target{Dir: "users", Filename: "user.go", Package: "users"},
				}},
			},
		}
		cmd := &cli.ExplainCommand{Config: cli.ExplainConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				gen,
				stubBackend{name: "be", lang: "stub"},
			},
			Selector: "users.User",
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
		out := stdout.String()
		if !strings.Contains(out, "Kind:") || !strings.Contains(out, "emit.struct") {
			t.Fatalf("expected Kind line for struct; got %q", out)
		}
	})

	t.Run("unresolved anchor exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t, "eidos")
		cmd := &cli.ExplainCommand{Config: cli.ExplainConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				stubBackend{name: "be", lang: "stub"},
			},
			Selector: "nope.Missing",
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitUserError {
			t.Fatalf("Execute = %d, want ExitUserError", code)
		}
		if !strings.Contains(stderr.String(), "resolves to no emit entity") {
			t.Fatalf("expected not-found diagnostic; got %q", stderr.String())
		}
	})
}

// TestExplainCommand_SlotSelector covers the slot form: anchor
// resolves to an entity that owns slots, modifier names a slot
// with at least one contribution.
func TestExplainCommand_SlotSelector(t *testing.T) {
	t.Parallel()

	t.Run("slot contributions print with set-by attribution", func(t *testing.T) {
		t.Parallel()
		env, stdout, _ := freshEnv(t, "eidos")
		host := &emit.Struct{
			Name: "User", Package: "users",
			Target: emit.Target{Dir: "users", Filename: "user.go", Package: "users"},
		}
		// Pre-seed a slot contribution so the generator runs and the
		// host carries one entry when Explain inspects it.
		_ = host.FieldsSlot().Append(
			&emit.Field{Name: "Email", Type: emit.Builtin("string")},
			emit.Provenance{SetBy: "validationgen"},
		)
		gen := emittingGenerator{
			name: "fixturegen",
			pkg: &emit.Package{
				Name: "users", Path: "users",
				Structs: []*emit.Struct{host},
			},
		}
		cmd := &cli.ExplainCommand{Config: cli.ExplainConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				gen,
				stubBackend{name: "be", lang: "stub"},
			},
			Selector: "users.User:fields",
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitOK {
			t.Fatalf("Execute = %d, want ExitOK", code)
		}
		if !strings.Contains(stdout.String(), "validationgen") {
			t.Fatalf("expected set-by attribution; got %q", stdout.String())
		}
	})

	t.Run("empty slot exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t, "eidos")
		gen := emittingGenerator{
			name: "fixturegen",
			pkg: &emit.Package{
				Name: "users", Path: "users",
				Structs: []*emit.Struct{{
					Name: "User", Package: "users",
					Target: emit.Target{Dir: "users", Filename: "user.go", Package: "users"},
				}},
			},
		}
		cmd := &cli.ExplainCommand{Config: cli.ExplainConfig{
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				gen,
				stubBackend{name: "be", lang: "stub"},
			},
			Selector: "users.User:fields",
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitUserError {
			t.Fatalf("Execute = %d, want ExitUserError", code)
		}
		if !strings.Contains(stderr.String(), "no contributions") {
			t.Fatalf("expected no-contributions diagnostic; got %q", stderr.String())
		}
	})
}

// TestParseSelector covers the parser's three forms plus the
// reject paths for empty inputs / missing modifiers. The parser
// is unexported so the test exercises it via Execute's
// command-line surface (an unparseable selector returns
// ExitUserError without running the pipeline).
func TestParseSelector(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		selector string
		wantCode int
		mustErr  string
	}{
		{
			name:     "empty selector",
			selector: "",
			wantCode: cli.ExitUserError,
			mustErr:  "selector is required",
		},
		{
			name:     "anchor missing in slot form",
			selector: ":fields",
			wantCode: cli.ExitUserError,
			mustErr:  "anchor and slot name are both required",
		},
		{
			name:     "modifier missing in meta form",
			selector: "users.User#",
			wantCode: cli.ExitUserError,
			mustErr:  "anchor and key are both required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env, _, stderr := freshEnv(t, "eidos")
			cmd := &cli.ExplainCommand{Config: cli.ExplainConfig{
				Plugins: []plugin.Plugin{
					stubFrontend{name: "fe"},
					stubBackend{name: "be", lang: "stub"},
				},
				Selector: tc.selector,
			}}
			code := cmd.Execute(t.Context(), env)
			if code != tc.wantCode {
				t.Fatalf("Execute = %d, want %d", code, tc.wantCode)
			}
			if !strings.Contains(stderr.String(), tc.mustErr) {
				t.Fatalf("stderr should contain %q; got %q", tc.mustErr, stderr.String())
			}
		})
	}
}

// TestExplainCommand_ConfigError covers the user-error path.
func TestExplainCommand_ConfigError(t *testing.T) {
	t.Parallel()

	t.Run("missing plugin exits ExitUserError", func(t *testing.T) {
		t.Parallel()
		env, _, stderr := freshEnv(t, "eidos")
		cfg := cli.DefaultConfig()
		cfg.Plugins = []cli.ConfigPlugin{{Name: "ghost"}}
		cmd := &cli.ExplainCommand{Config: cli.ExplainConfig{
			File: cfg,
			Plugins: []plugin.Plugin{
				stubFrontend{name: "fe"},
				stubBackend{name: "be", lang: "stub"},
			},
			Selector: "anything",
		}}
		if code := cmd.Execute(t.Context(), env); code != cli.ExitUserError {
			t.Fatalf("Execute = %d, want ExitUserError", code)
		}
		if !strings.Contains(stderr.String(), "ghost") {
			t.Fatalf("stderr should name the missing plugin; got %q", stderr.String())
		}
	})
}

// errSentinelImport keeps the errors import referenced in case
// further test helpers grow.
var _ = errors.New
