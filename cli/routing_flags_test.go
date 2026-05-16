// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"errors"
	"flag"
	"reflect"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
)

// Test list for RoutingFlags:
//
//  1. Apply on a fresh Builder threads -target to WithTargetSymbol
//     so Pipeline.TargetSymbol returns it.
//  2. Apply threads -o to WithOutputFilename.
//  3. Apply threads -p to WithOutputPackage so the default
//     LayoutPolicy carries the value with LayerCLI attribution.
//  4. Apply threads -layout to WithOutputLayout.
//  5. Apply threads -output-dir to WithOutputDir.
//  6. Empty fields short-circuit — no setter is invoked.
//  7. Register binds every flag onto the supplied FlagSet under
//     the documented FlagTarget / FlagOutput / FlagPackage /
//     FlagLayout / FlagOutputDir names.
//  8. Infer derives -target from $GOFILE when env var set and
//     Target is empty.
//  9. Infer leaves explicit -target untouched even when $GOFILE
//     is set.
// 10. Validate rejects -o without -target after inference.
// 11. Validate rejects -layout centralised without -p and without
//     project-level output.package.
// 12. Validate accepts -layout centralised when output.package
//     resolves from config.
// 13. Validate surfaces a warning for -output-dir without
//     -layout centralised.

// TestRoutingFlags_Apply_OutputScopes pins the scoped `-o` shapes
// introduced for multi-output plugins: `-o <plugin>=<path>` pins
// the named plugin's primary output; `-o <plugin>:<tag>=<path>`
// pins one specific tagged output. Unscoped `-o <path>` continues
// to map to the legacy global pinning. Multiple `-o` flags
// accumulate; each entry routes to the matching builder setter
// based on its scope shape.
func TestRoutingFlags_Apply_OutputScopes(t *testing.T) {
	t.Parallel()

	t.Run("unscoped -o threads to WithOutputFilename", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Output: []string{"gen.go"}}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got, want := p.OutputFilename(), "gen.go"; got != want {
			t.Fatalf("OutputFilename = %q, want %q", got, want)
		}
	})

	t.Run("`<plugin>=<path>` threads to per-plugin override", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Output: []string{"mock=mocks/handler.go"}}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		got, ok := p.PluginOutputFilename("mock", "")
		if !ok || got != "mocks/handler.go" {
			t.Fatalf("PluginOutputFilename(mock, \"\") = %q, %v; want mocks/handler.go, true", got, ok)
		}
	})

	t.Run("`<plugin>:<tag>=<path>` threads to per-(plugin, tag) override", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Output: []string{"mock:test=tests/handler.go"}}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		got, ok := p.PluginOutputFilename("mock", "test")
		if !ok || got != "tests/handler.go" {
			t.Fatalf("PluginOutputFilename(mock, test) = %q, %v; want tests/handler.go, true", got, ok)
		}
	})

	t.Run("multiple -o flags accumulate by shape", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Output: []string{
			"mock=mocks/handler.go",
			"mock:test=tests/handler.go",
		}}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got, ok := p.PluginOutputFilename("mock", ""); !ok || got != "mocks/handler.go" {
			t.Fatalf("PluginOutputFilename(mock, \"\") = %q, %v", got, ok)
		}
		if got, ok := p.PluginOutputFilename("mock", "test"); !ok || got != "tests/handler.go" {
			t.Fatalf("PluginOutputFilename(mock, test) = %q, %v", got, ok)
		}
	})

	t.Run("`<plugin>=<path>` with `:` in the path parses unambiguously", func(t *testing.T) {
		t.Parallel()
		// The `:` separator lives inside the key portion; once we
		// see `=`, the remainder is the path verbatim and may
		// contain colons (e.g. on Windows-style or remote paths).
		flags := cli.RoutingFlags{Output: []string{"mock:test=tests/handler.go:1"}}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		got, ok := p.PluginOutputFilename("mock", "test")
		if !ok || got != "tests/handler.go:1" {
			t.Fatalf("PluginOutputFilename(mock, test) = %q, %v; want tests/handler.go:1, true", got, ok)
		}
	})
}

// TestRoutingFlags_Apply_ThreadsToBuilder pins the contract: each
// flag value, when non-empty, lands on the corresponding
// Builder.With* setter so the constructed Pipeline carries the
// expected accessor / policy values.
func TestRoutingFlags_Apply_ThreadsToBuilder(t *testing.T) {
	t.Parallel()

	t.Run("Target threads to Pipeline.TargetSymbol", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Target: "Foo"}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got, want := p.TargetSymbol(), "Foo"; got != want {
			t.Fatalf("TargetSymbol = %q, want %q", got, want)
		}
	})

	t.Run("Output threads to Pipeline.OutputFilename", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Output: []string{"gen.go"}}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got, want := p.OutputFilename(), "gen.go"; got != want {
			t.Fatalf("OutputFilename = %q, want %q", got, want)
		}
	})

	t.Run("Package threads to LayoutPolicy with LayerCLI attribution", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Package: "generated"}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		got := p.LayoutPolicyFor("any")
		if got.Package != "generated" || got.PackageFrom != manifest.LayerCLI {
			t.Fatalf("Package = %q from %q, want generated from cli", got.Package, got.PackageFrom)
		}
	})

	t.Run("Layout threads to LayoutPolicy with LayerCLI attribution", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Layout: pipeline.LayoutCentralised}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		got := p.LayoutPolicyFor("any")
		if got.Layout != pipeline.LayoutCentralised || got.LayoutFrom != manifest.LayerCLI {
			t.Fatalf("Layout = %q from %q, want centralised from cli", got.Layout, got.LayoutFrom)
		}
	})

	t.Run("OutputDir threads to LayoutPolicy with LayerCLI attribution", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{OutputDir: "internal/gen"}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		got := p.LayoutPolicyFor("any")
		if got.Dir != "internal/gen" || got.DirFrom != manifest.LayerCLI {
			t.Fatalf("Dir = %q from %q, want internal/gen from cli", got.Dir, got.DirFrom)
		}
	})

	t.Run("empty RoutingFlags leaves the framework defaults in place", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{}
		b := pipeline.New().
			WithFrontend(stubFrontend{name: "fe"}).
			WithBackend(stubBackend{name: "be", lang: "stub"})
		flags.Apply(b)
		p, err := b.Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		got := p.LayoutPolicyFor("any")
		if got.LayoutFrom != manifest.LayerFramework {
			t.Fatalf("LayoutFrom = %q, want framework (no flag set)", got.LayoutFrom)
		}
	})
}

// TestRoutingFlags_Register binds the routing flags onto a stock
// flag.FlagSet under the documented names so consumer binaries
// reaching for the same names see them register cleanly.
func TestRoutingFlags_Register(t *testing.T) {
	t.Parallel()

	t.Run("Register binds every routing flag onto the supplied FlagSet", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var flags cli.RoutingFlags
		flags.Register(fs)
		for _, name := range []string{
			cli.FlagTarget, cli.FlagOutput, cli.FlagPackage,
			cli.FlagLayout, cli.FlagOutputDir,
		} {
			if fs.Lookup(name) == nil {
				t.Errorf("FlagSet missing flag %q after Register", name)
			}
		}
	})

	t.Run("Parsed flag values land on the RoutingFlags receiver", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var flags cli.RoutingFlags
		flags.Register(fs)
		args := []string{
			"-" + cli.FlagTarget, "Article",
			"-" + cli.FlagOutput, "article_gen.go",
			"-" + cli.FlagPackage, "generated",
			"-" + cli.FlagLayout, pipeline.LayoutCentralised,
			"-" + cli.FlagOutputDir, "internal/gen",
		}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("Parse: %v", err)
		}
		want := cli.RoutingFlags{
			Target:    "Article",
			Output:    []string{"article_gen.go"},
			Package:   "generated",
			Layout:    pipeline.LayoutCentralised,
			OutputDir: "internal/gen",
		}
		if !reflect.DeepEqual(flags, want) {
			t.Fatalf("RoutingFlags after Parse = %+v, want %+v", flags, want)
		}
	})

	t.Run("repeated -o flags accumulate into the slice", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var flags cli.RoutingFlags
		flags.Register(fs)
		args := []string{
			"-" + cli.FlagOutput, "mock=mocks/handler.go",
			"-" + cli.FlagOutput, "mock:test=tests/handler.go",
		}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("Parse: %v", err)
		}
		want := []string{"mock=mocks/handler.go", "mock:test=tests/handler.go"}
		if !reflect.DeepEqual(flags.Output, want) {
			t.Fatalf("Output after Parse = %+v, want %+v", flags.Output, want)
		}
	})
}

// TestRoutingFlags_Infer_GOFILE pins the inference contract:
// when GOFILE is set and Target is empty, Infer derives Target
// from the basename of $GOFILE (minus the `.go` extension);
// when Target is explicit, the inference is a no-op. The tests
// inject the env via a captured-getenv closure so parallel
// subtests don't fight over the process's environment.
func TestRoutingFlags_Infer_GOFILE(t *testing.T) {
	t.Parallel()

	fakeEnv := func(v string) func(string) string {
		return func(key string) string {
			if key == "GOFILE" {
				return v
			}
			return ""
		}
	}

	t.Run("GOFILE basename without .go becomes Target when Target is empty", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{}
		flags.Infer(fakeEnv("article.go"))
		if got, want := flags.Target, "article"; got != want {
			t.Fatalf("Target after Infer = %q, want %q", got, want)
		}
	})

	t.Run("Explicit Target wins over GOFILE", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Target: "Comment"}
		flags.Infer(fakeEnv("article.go"))
		if got, want := flags.Target, "Comment"; got != want {
			t.Fatalf("Target after Infer = %q, want %q (explicit wins)", got, want)
		}
	})

	t.Run("Unset GOFILE leaves Target empty", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{}
		flags.Infer(fakeEnv(""))
		if flags.Target != "" {
			t.Fatalf("Target after Infer = %q, want empty (no GOFILE)", flags.Target)
		}
	})

	t.Run("GOFILE without .go extension is used verbatim", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{}
		flags.Infer(fakeEnv("Article"))
		if got, want := flags.Target, "Article"; got != want {
			t.Fatalf("Target after Infer = %q, want %q (no .go to strip)", got, want)
		}
	})

	t.Run("Infer reports whether the inference fired", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{}
		if !flags.Infer(fakeEnv("article.go")) {
			t.Fatalf("Infer should report true when it fired")
		}
		flags = cli.RoutingFlags{Target: "X"}
		if flags.Infer(fakeEnv("article.go")) {
			t.Fatalf("Infer should report false when Target was already set")
		}
	})
}

// TestRoutingFlags_Validate covers the run-time validation
// rules: -o without -target rejected, -layout centralised
// without a resolvable package rejected, -output-dir without
// centralised layout surfaces a warning.
func TestRoutingFlags_Validate(t *testing.T) {
	t.Parallel()

	t.Run("unscoped -o without -target is rejected", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Output: []string{"x.go"}}
		_, err := flags.Validate(cli.DefaultConfig())
		var ce *cli.ConfigError
		if !errsAs(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %v", err)
		}
		if !strings.Contains(ce.Reason, "-"+cli.FlagOutput) {
			t.Fatalf("error should name -o; got %q", ce.Reason)
		}
	})

	t.Run("-layout centralised without -p or output.package is rejected", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Layout: pipeline.LayoutCentralised}
		_, err := flags.Validate(cli.DefaultConfig())
		var ce *cli.ConfigError
		if !errsAs(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %v", err)
		}
		if !strings.Contains(ce.Reason, "-"+cli.FlagLayout) {
			t.Fatalf("error should name -layout; got %q", ce.Reason)
		}
	})

	t.Run("-layout centralised with -p validates clean", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Layout: pipeline.LayoutCentralised, Package: "gen"}
		warnings, err := flags.Validate(cli.DefaultConfig())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 0 {
			t.Fatalf("unexpected warnings: %v", warnings)
		}
	})

	t.Run("-layout centralised with output.package from config validates clean", func(t *testing.T) {
		t.Parallel()
		cfg := cli.DefaultConfig()
		cfg.Output = cli.ConfigOutput{Package: "gen"}
		flags := cli.RoutingFlags{Layout: pipeline.LayoutCentralised}
		warnings, err := flags.Validate(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 0 {
			t.Fatalf("unexpected warnings: %v", warnings)
		}
	})

	t.Run("-output-dir without centralised surfaces a warning", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Target: "X", OutputDir: "internal/gen"}
		warnings, err := flags.Validate(cli.DefaultConfig())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 1 || !strings.Contains(warnings[0], "-"+cli.FlagOutputDir) {
			t.Fatalf("expected -output-dir warning; got %v", warnings)
		}
	})

	t.Run("unknown layout value is rejected", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Layout: "weird"}
		_, err := flags.Validate(cli.DefaultConfig())
		var ce *cli.ConfigError
		if !errsAs(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %v", err)
		}
	})

	t.Run("scoped -o `<plugin>=<path>` does not require -target", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Output: []string{"mock=mocks/handler.go"}}
		warnings, err := flags.Validate(cli.DefaultConfig())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 0 {
			t.Fatalf("unexpected warnings: %v", warnings)
		}
	})

	t.Run("scoped -o `<plugin>:<tag>=<path>` does not require -target", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{Output: []string{"mock:test=tests/handler.go"}}
		warnings, err := flags.Validate(cli.DefaultConfig())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(warnings) != 0 {
			t.Fatalf("unexpected warnings: %v", warnings)
		}
	})

	t.Run("malformed -o value is rejected", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{
			"=path.go",     // empty key
			"plugin=",      // empty path
			"plugin:=p.go", // empty tag
			":tag=p.go",    // empty plugin
		} {
			flags := cli.RoutingFlags{Output: []string{value}}
			_, err := flags.Validate(cli.DefaultConfig())
			var ce *cli.ConfigError
			if !errsAs(err, &ce) {
				t.Errorf("%q: expected *cli.ConfigError; got %v", value, err)
				continue
			}
			if !strings.Contains(ce.Reason, "-"+cli.FlagOutput) {
				t.Errorf("%q: error should name -o; got %q", value, ce.Reason)
			}
		}
	})

	t.Run("multiple unscoped -o values are rejected", func(t *testing.T) {
		t.Parallel()
		flags := cli.RoutingFlags{
			Target: "X",
			Output: []string{"a.go", "b.go"},
		}
		_, err := flags.Validate(cli.DefaultConfig())
		var ce *cli.ConfigError
		if !errsAs(err, &ce) {
			t.Fatalf("expected *cli.ConfigError; got %v", err)
		}
		if !strings.Contains(ce.Reason, "unscoped") {
			t.Fatalf("error should mention `unscoped`; got %q", ce.Reason)
		}
	})
}

// errsAs is the standard errors.As entry typed for the test
// assertions — keeps each Validate-error subtest terse without
// repeating the import + var declaration at every site.
func errsAs[T error](err error, target *T) bool {
	return errors.As(err, target)
}

// pluginPkg keeps the linter happy: plugin is imported above so
// stubFrontend / stubBackend literals compile; this references it
// once if the import would otherwise be unused after refactors.
var _ = plugin.Generator(nil)
