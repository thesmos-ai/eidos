// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/pipeline"
)

// RoutingFlags is the bundle of routing-layer CLI flag values
// shared across [RunCommand], [CheckCommand], [PlanCommand],
// [ExplainCommand], and [PruneCommand]. Each command embeds
// this struct in its Config so the flag wiring is uniform:
// [RoutingFlags.Register] binds the five flags onto the command's
// [flag.FlagSet]; [RoutingFlags.Infer] derives -target from the
// `GOFILE` env var when invoked through `//go:generate`;
// [RoutingFlags.Validate] enforces the documented combination
// rules against the merged config-and-flag state;
// [RoutingFlags.Apply] threads the resolved values onto a
// [pipeline.Builder] before [pipeline.Builder.Build] runs.
//
// Empty fields short-circuit at every step — `Apply` skips the
// matching Builder setter, `Infer` only fires when GOFILE is set
// and Target is empty, and `Validate` only checks rules whose
// triggering field is non-empty. The zero value is a no-op
// across the lifecycle.
type RoutingFlags struct {
	// Target restricts the run to source decls whose unqualified
	// name equals the value, or whose qualified name ends with
	// `.<value>`. The disambiguator for projects with the same
	// decl name in multiple packages: `pkg.Foo` selects only the
	// Foo in package pkg.
	Target string

	// Output captures every `-o` flag value passed on the CLI.
	// Each entry takes one of three shapes:
	//
	//   - `<path>` — unscoped legacy form. Pins
	//     [emit.Target.Filename] for every emitted decl in scope.
	//     Requires `-target` so the scope is non-ambiguous. Maps
	//     to [pipeline.Builder.WithOutputFilename]. At most one
	//     unscoped entry is permitted across the slice.
	//
	//   - `<plugin>=<path>` — per-plugin pin. Pins Filename for
	//     decls emitted by the named plugin (across every output
	//     the plugin declares). Maps to
	//     [pipeline.Builder.WithPluginOutputFilename] with empty
	//     tag. No `-target` requirement: the plugin filter is
	//     itself the scope.
	//
	//   - `<plugin>:<tag>=<path>` — per-(plugin, tag) pin. Pins
	//     Filename for the named plugin's named output only;
	//     other outputs from the same plugin route per their
	//     declared suffix. Maps to
	//     [pipeline.Builder.WithPluginOutputFilename] with the
	//     supplied tag. No `-target` requirement.
	//
	// The `:` separator lives inside the key portion of the
	// `<key>=<path>` shape, so the path may contain `:` without
	// ambiguity. Multiple `-o` flags accumulate; specificity at
	// routing time is (plugin, tag) > (plugin, "") > unscoped.
	Output []string

	// Package pins [emit.Target.Package] for every emitted decl
	// in scope. Maps to [pipeline.Builder.WithOutputPackage].
	Package string

	// Layout overrides the project-default layout selector.
	// Accepts [pipeline.LayoutAlongsideSource] or
	// [pipeline.LayoutCentralised]. Maps to
	// [pipeline.Builder.WithOutputLayout].
	Layout string

	// OutputDir sets the rendered output directory under
	// centralised layout. Maps to
	// [pipeline.Builder.WithOutputDir].
	OutputDir string
}

// Register binds every routing-layer flag onto fs using the
// canonical [FlagTarget] / [FlagOutput] / [FlagPackage] /
// [FlagLayout] / [FlagOutputDir] names and the documented usage
// strings. The `-o` flag accepts multiple values via repetition;
// every other flag is single-valued.
func (rf *RoutingFlags) Register(fs *flag.FlagSet) {
	fs.StringVar(&rf.Target, FlagTarget, "", UsageTarget)
	fs.Var((*outputFlag)(&rf.Output), FlagOutput, UsageOutput)
	fs.StringVar(&rf.Package, FlagPackage, "", UsagePackage)
	fs.StringVar(&rf.Layout, FlagLayout, "", UsageLayout)
	fs.StringVar(&rf.OutputDir, FlagOutputDir, "", UsageOutputDir)
}

// outputFlag is the [flag.Value] adapter that accumulates every
// `-o <value>` invocation into the receiver slice. Set is invoked
// once per `-o` flag the user passes; the slice preserves order
// so the Apply / Validate stages see entries in CLI declaration
// order.
type outputFlag []string

// Set appends value to the receiver slice. The flag package
// guarantees one Set per invocation.
func (f *outputFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// String returns the comma-joined slice contents for `flag` package
// introspection; CLI users never see this — repeated `-o` flags
// stay distinct on the command line.
func (f *outputFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

// outputSpec captures one parsed `-o` flag value. Plugin and Tag
// are both empty for the legacy unscoped form; Plugin alone is
// set for the `<plugin>=<path>` form; both are set for the
// `<plugin>:<tag>=<path>` form.
type outputSpec struct {
	Plugin string
	Tag    string
	Path   string
}

// parseOutputSpec classifies one `-o` flag value into the legacy
// unscoped form (no `=`) or one of the scoped forms. The first
// `=` separates the optional `<plugin>` or `<plugin>:<tag>` key
// from the path; the first `:` within the key separates plugin
// from tag. Values may carry `:` in the path verbatim — once the
// `=` separator fires, every subsequent character is the path.
//
// Returns a populated [outputSpec] and ok=true on success. An
// empty value, a value with empty plugin name (`=path`), or a
// value with empty tag (`plugin:=path`) returns ok=false; the
// caller surfaces a config error citing the offending input.
func parseOutputSpec(value string) (outputSpec, bool) {
	if value == "" {
		return outputSpec{}, false
	}
	key, path, hasEq := strings.Cut(value, "=")
	if !hasEq {
		// Unscoped legacy form.
		return outputSpec{Path: value}, true
	}
	if path == "" {
		return outputSpec{}, false
	}
	if key == "" {
		// `=path` with no key — reject; ambiguous between the
		// unscoped form and a typo of the scoped form.
		return outputSpec{}, false
	}
	if plugin, tag, hasColon := strings.Cut(key, ":"); hasColon {
		if plugin == "" || tag == "" {
			return outputSpec{}, false
		}
		return outputSpec{Plugin: plugin, Tag: tag, Path: path}, true
	}
	return outputSpec{Plugin: key, Path: path}, true
}

// Apply threads the configured flag values onto b. Each non-empty
// field invokes the matching [pipeline.Builder.With*] setter;
// empty fields short-circuit so the framework default / lower-
// priority precedence layer remains in place. Output entries are
// routed by their parsed shape: unscoped → WithOutputFilename;
// scoped → WithPluginOutputFilename.
func (rf RoutingFlags) Apply(b *pipeline.Builder) {
	if rf.Target != "" {
		b.WithTargetSymbol(rf.Target)
	}
	for _, value := range rf.Output {
		spec, ok := parseOutputSpec(value)
		if !ok {
			continue
		}
		if spec.Plugin == "" {
			b.WithOutputFilename(spec.Path)
			continue
		}
		b.WithPluginOutputFilename(spec.Plugin, spec.Tag, spec.Path)
	}
	if rf.Package != "" {
		b.WithOutputPackage(rf.Package)
	}
	if rf.Layout != "" {
		b.WithOutputLayout(rf.Layout)
	}
	if rf.OutputDir != "" {
		b.WithOutputDir(rf.OutputDir)
	}
}

// Infer derives Target from the GOFILE environment variable
// when GOFILE is set and Target is empty — the canonical
// `//go:generate eidos run` flow where the surrounding source
// file pins scope implicitly. The inferred value is the GOFILE
// basename with any trailing `.go` extension stripped.
//
// getenv is the environment accessor (typically [os.Getenv]);
// injected so tests pin the environment without touching the
// process's actual env state.
//
// Returns true when inference fired (Target was updated), false
// when Target was already set or GOFILE was unset / empty.
func (rf *RoutingFlags) Infer(getenv func(string) string) bool {
	if rf.Target != "" {
		return false
	}
	gofile := getenv("GOFILE")
	if gofile == "" {
		return false
	}
	rf.Target = strings.TrimSuffix(gofile, ".go")
	return true
}

// Validate enforces the spec's combination rules against the
// merged flag-and-config state. cfg supplies the project-level
// config-file values that complement the flag inputs (notably
// `output.package` which can satisfy the centralised-requires-
// package rule without `-p`). Returns warnings for non-fatal
// advisories and a *[ConfigError] on the first hard violation.
//
// Rules:
//   - Layout must be one of the documented enum values.
//   - Every `-o` value parses as one of the documented shapes
//     (unscoped `<path>`, `<plugin>=<path>`, or
//     `<plugin>:<tag>=<path>`); malformed values are rejected.
//   - At most one unscoped `-o` value is permitted across the
//     slice — the legacy global pinning is single-valued.
//   - An unscoped `-o` without `-target` (post-Infer) is rejected:
//     the unscoped form pins one filename for every emitted decl
//     in scope, and pairing it with multi-symbol scope produces
//     undefined per-decl behaviour. Scoped `-o` values carry the
//     scope on themselves (the plugin filter, optionally narrowed
//     by tag) and don't require `-target`.
//   - Layout = centralised requires a resolvable Package (-p OR
//     project-level output.package).
//   - OutputDir without Layout = centralised is a warning —
//     alongside-source derives Dir from origin and ignores the
//     flag.
func (rf RoutingFlags) Validate(cfg *Config) ([]string, error) {
	if err := validateLayoutEnum("-"+FlagLayout, rf.Layout, ""); err != nil {
		return nil, err
	}
	unscopedCount := 0
	for _, value := range rf.Output {
		spec, ok := parseOutputSpec(value)
		if !ok {
			return nil, &ConfigError{
				Reason: fmt.Sprintf(
					"-%s value %q: expected <path>, <plugin>=<path>, or <plugin>:<tag>=<path>",
					FlagOutput, value,
				),
			}
		}
		if spec.Plugin == "" {
			unscopedCount++
		}
	}
	if unscopedCount > 1 {
		return nil, &ConfigError{
			Reason: fmt.Sprintf(
				"-%s: at most one unscoped value is permitted; scoped forms (<plugin>=<path>, <plugin>:<tag>=<path>) may repeat",
				FlagOutput,
			),
		}
	}
	if unscopedCount == 1 && rf.Target == "" {
		return nil, &ConfigError{
			Reason: fmt.Sprintf(
				"-%s without a scope (-%s or <plugin>=<path>): pinning a single filename without a scope filter produces undefined per-decl behaviour",
				FlagOutput,
				FlagTarget,
			),
		}
	}
	if rf.Layout == pipeline.LayoutCentralised {
		effPackage := rf.Package
		if effPackage == "" && cfg != nil {
			effPackage = cfg.Output.Package
		}
		if effPackage == "" {
			return nil, &ConfigError{
				Reason: fmt.Sprintf(
					"-%s=%s requires -%s or project-level output.package",
					FlagLayout, pipeline.LayoutCentralised, FlagPackage,
				),
			}
		}
	}
	var warnings []string
	if rf.OutputDir != "" && rf.Layout != pipeline.LayoutCentralised {
		warnings = append(warnings,
			"-"+FlagOutputDir+" is set but -"+FlagLayout+
				" is not centralised; the directory will be ignored at routing time")
	}
	return warnings, nil
}

// Resolve runs the documented routing-flag lifecycle in order:
// $GOFILE inference fills Target when GOFILE is set and Target
// was not explicitly passed; validation enforces the spec rules
// against the merged flag-and-config state; warnings are emitted
// to env.Diag under the "cli.routing" attribution at Warn
// severity. Returns the resolved flags and a *[ConfigError]
// when validation rejects the combination — callers exit
// [ExitUserError] before any pipeline construction work.
//
// verbose enables the GOFILE-inference Info diagnostic so users
// running with `-verbose` see the inferred scope alongside their
// other flag-level traces.
func (rf RoutingFlags) Resolve(env *Env, cfg *Config, verbose bool) (RoutingFlags, error) {
	resolved := rf
	inferred := resolved.Infer(os.Getenv)
	if inferred && verbose && env != nil && env.Diag != nil {
		ps := env.Diag.For("cli.routing")
		ps.Infof(position.Pos{},
			"-%s inferred from $GOFILE: %q", FlagTarget, resolved.Target)
	}
	warnings, err := resolved.Validate(cfg)
	if err != nil {
		return resolved, err
	}
	if env != nil && env.Diag != nil {
		ps := env.Diag.For("cli.routing")
		for _, w := range warnings {
			ps.Warnf(position.Pos{}, "%s", w)
		}
	}
	return resolved, nil
}
