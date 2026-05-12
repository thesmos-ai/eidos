// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"flag"

	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
)

// RunConfig holds the inputs for [RunCommand]. The command
// constructs a pipeline from File + Plugins, applies the runtime
// flag overrides, runs it against the supplied patterns, and
// reports the result.
type RunConfig struct {
	// File is the loaded config. May be nil; the default applies.
	File *Config

	// Plugins is the consumer's static plugin universe.
	Plugins []plugin.Plugin

	// Patterns are the positional input descriptors passed to each
	// frontend (typically Go-style import paths). Empty falls back
	// to the file's source patterns.
	Patterns []string

	// NoCache disables the build cache for this invocation, winning
	// over the file's cache.enabled.
	NoCache bool

	// DiagFormat selects the diagnostic rendering format.
	DiagFormat DiagFormat

	// Verbose surfaces Info diagnostics.
	Verbose bool

	// Quiet suppresses Warn diagnostics.
	Quiet bool

	// Routing carries the run's routing-layer flag overrides
	// (`-target` / `-o` / `-p` / `-layout` / `-output-dir`). The
	// values flow into the pipeline's Builder via
	// [RoutingFlags.Apply] before [pipeline.Builder.Build] runs.
	Routing RoutingFlags
}

// RunCommand executes the configured pipeline and writes output
// through its sink. Returns:
//
//   - [ExitOK] on success.
//   - [ExitUserError] on configuration faults.
//   - [ExitPipelineError] when any plugin emitted an Error
//     diagnostic, or when the pipeline returned a non-Run error
//     (e.g. ErrNoSink, ErrNoFrontend).
//   - [ExitInternalError] on a recovered panic.
//
// The supplied context threads to [pipeline.Pipeline.Run] for
// cancellation.
type RunCommand struct{ Config RunConfig }

// RegisterFlags binds [RunCommand]'s flags into fs.
func (c *RunCommand) RegisterFlags(fs *flag.FlagSet) {
	fs.Var(&c.Config.DiagFormat, FlagDiagFormat, UsageDiagFormat)
	fs.BoolVar(&c.Config.Verbose, FlagVerbose, false, UsageVerbose)
	fs.BoolVar(&c.Config.Quiet, FlagQuiet, false, UsageQuiet)
	fs.BoolVar(&c.Config.NoCache, FlagNoCache, false, UsageNoCache)
	c.Config.Routing.Register(fs)
}

// Execute runs the pipeline.
func (c *RunCommand) Execute(ctx context.Context, env *Env) (exit int) {
	defer recoverInto(env, &exit)

	cfg := c.Config.File
	if cfg == nil {
		cfg = DefaultConfig()
	}
	routing, err := c.Config.Routing.Resolve(env, cfg, c.Config.Verbose)
	if err != nil {
		writeErr(env, "%v", err)
		return ExitUserError
	}
	p, err := buildPipeline(env, cfg, c.Config.Plugins, pipelineOverride{
		NoCache: c.Config.NoCache,
		Verbose: c.Config.Verbose,
		Routing: routing,
	})
	if err != nil {
		writeErr(env, "%v", err)
		return ExitUserError
	}
	patterns := c.Config.Patterns
	if len(patterns) == 0 {
		patterns = patternsFromConfig(cfg)
	}
	runErr := p.Run(ctx, patterns...)
	rerr := RenderDiagnostics(env.Stderr, p.Diag(), c.Config.DiagFormat, c.Config.Verbose, c.Config.Quiet)
	if rerr != nil {
		writeErr(env, "%v", rerr)
	}
	if runErr == nil {
		return ExitOK
	}
	if errors.Is(runErr, pipeline.ErrRunHadErrors) {
		return ExitPipelineError
	}
	writeErr(env, "%v", runErr)
	return ExitPipelineError
}
