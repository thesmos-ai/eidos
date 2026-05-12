// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// CheckConfig holds the inputs for [CheckCommand]. The command
// runs the pipeline against an in-memory sink, then byte-equal
// compares every captured output against the corresponding file
// on disk. Any difference — including whitespace — counts as
// drift.
type CheckConfig struct {
	File       *Config
	Plugins    []plugin.Plugin
	DiagFormat DiagFormat
	Verbose    bool
	Quiet      bool
}

// CheckCommand implements the CI-gate `check` semantic. Returns:
//
//   - [ExitOK] when no drift detected.
//   - [ExitCheckDrift] when one or more files differ from disk.
//   - [ExitUserError] on configuration faults.
//   - [ExitPipelineError] when the pipeline run emitted Error
//     diagnostics.
//   - [ExitInternalError] on a recovered panic.
type CheckCommand struct{ Config CheckConfig }

// RegisterFlags binds [CheckCommand]'s flags into fs.
func (c *CheckCommand) RegisterFlags(fs *flag.FlagSet) {
	fs.Var(&c.Config.DiagFormat, FlagDiagFormat, UsageDiagFormat)
	fs.BoolVar(&c.Config.Verbose, FlagVerbose, false, UsageVerbose)
	fs.BoolVar(&c.Config.Quiet, FlagQuiet, false, UsageQuiet)
}

// Execute runs the pipeline through an in-memory sink and reports
// any byte-level drift against the on-disk state.
func (c *CheckCommand) Execute(ctx context.Context, env *Env) (exit int) {
	defer recoverInto(env, &exit)

	cfg := c.Config.File
	if cfg == nil {
		cfg = DefaultConfig()
	}
	memSink := sink.NewMemory()
	p, err := buildPipeline(env, cfg, c.Config.Plugins, pipelineOverride{
		Verbose:      c.Config.Verbose,
		SinkOverride: memSink,
	})
	if err != nil {
		writeErr(env, "%v", err)
		return ExitUserError
	}
	runErr := p.Run(ctx, patternsOrDefault(cfg)...)
	rerr := RenderDiagnostics(env.Stderr, p.Diag(), c.Config.DiagFormat, c.Config.Verbose, c.Config.Quiet)
	if rerr != nil {
		writeErr(env, "%v", rerr)
	}
	if runErr != nil && !errors.Is(runErr, pipeline.ErrRunHadErrors) {
		writeErr(env, "%v", runErr)
		return ExitPipelineError
	}
	if errors.Is(runErr, pipeline.ErrRunHadErrors) {
		return ExitPipelineError
	}
	return c.reportDrift(env, memSink.Files())
}

// reportDrift compares every (target, body) pair the in-memory
// sink captured against the corresponding file on disk under
// env.Workdir. Differences print to env.Stdout in sorted target
// order so the report is deterministic across runs.
func (*CheckCommand) reportDrift(env *Env, current map[emit.Target][]byte) int {
	targets := make([]emit.Target, 0, len(current))
	for t := range current {
		targets = append(targets, t)
	}
	sort.Slice(targets, func(i, j int) bool {
		return joinTarget(targets[i]) < joinTarget(targets[j])
	})

	drifted := 0
	for _, t := range targets {
		path := filepath.Join(env.Workdir, t.Dir, t.Filename)
		disk, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(env.Stdout, "drift: %s (missing from disk)\n", path)
			drifted++
			continue
		}
		if err != nil {
			writeErr(env, "read %s: %v", path, err)
			drifted++
			continue
		}
		if !bytes.Equal(disk, current[t]) {
			fmt.Fprintf(env.Stdout, "drift: %s (content differs)\n", path)
			drifted++
		}
	}
	if drifted == 0 {
		fmt.Fprintf(env.Stdout, "check: no drift detected across %d output(s)\n", len(targets))
		return ExitOK
	}
	fmt.Fprintf(env.Stdout, "check: %d drifting file(s)\n", drifted)
	return ExitCheckDrift
}

// joinTarget returns the on-disk-relative path for a target —
// "<Dir>/<Filename>" — used to sort drift reports deterministically.
func joinTarget(t emit.Target) string {
	return filepath.Join(t.Dir, t.Filename)
}
