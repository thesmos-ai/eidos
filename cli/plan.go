// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"flag"
	"fmt"

	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
)

// PlanConfig holds the inputs for [PlanCommand]. The command
// constructs a pipeline from the supplied universe + config and
// prints the resolved [pipeline.Plan] — every frontend, annotator,
// generator in execution order plus the backend. No pipeline IO
// happens.
type PlanConfig struct {
	// File is the loaded config. The plan reflects whichever
	// plugins File.Plugins enables.
	File *Config

	// Plugins is the consumer's static plugin universe.
	Plugins []plugin.Plugin

	// Format selects text (default) or JSON output.
	Format DiagFormat
}

// PlanCommand prints the resolved plugin order without running
// the pipeline. Useful for CI debugging ("what will run?") and
// for documentation.
type PlanCommand struct{ Config PlanConfig }

// RegisterFlags binds [PlanCommand]'s flags into fs.
func (c *PlanCommand) RegisterFlags(fs *flag.FlagSet) {
	fs.Var(&c.Config.Format, FlagDiagFormat, UsageDiagFormat)
}

// Execute resolves the plan and prints it. Returns [ExitUserError]
// on configuration faults; otherwise [ExitOK].
func (c *PlanCommand) Execute(_ context.Context, env *Env) int {
	cfg := c.Config.File
	if cfg == nil {
		cfg = DefaultConfig()
	}
	p, err := buildPipeline(env, cfg, c.Config.Plugins, pipelineOverride{})
	if err != nil {
		writeErr(env, "%v", err)
		return ExitUserError
	}
	plan := p.Plan()
	switch c.Config.Format {
	case DiagFormatJSON:
		return c.writeJSON(env, plan)
	default:
		return c.writeText(env, plan)
	}
}

// writeText renders the plan as a human-readable block grouped by
// phase. Bucket boundaries surface so callers can see priority
// groupings at a glance.
func (*PlanCommand) writeText(env *Env, plan *pipeline.Plan) int {
	w := env.Stdout
	fmt.Fprintln(w, "frontends:")
	for _, fe := range plan.Frontends {
		fmt.Fprintf(w, "  - %s\n", fe.Name())
	}
	fmt.Fprintln(w, "annotators:")
	for _, bucket := range plan.AnnotatorBuckets {
		fmt.Fprintf(w, "  bucket %d:\n", int(bucket.Priority))
		for _, ann := range bucket.Plugins {
			fmt.Fprintf(w, "    - %s\n", ann.Name())
		}
	}
	fmt.Fprintln(w, "generators:")
	for _, bucket := range plan.GeneratorBuckets {
		fmt.Fprintf(w, "  bucket %d:\n", int(bucket.Priority))
		for _, gen := range bucket.Plugins {
			fmt.Fprintf(w, "    - %s\n", gen.Name())
		}
	}
	if plan.Backend != nil {
		fmt.Fprintf(w, "backend: %s\n", plan.Backend.Name())
	} else {
		fmt.Fprintln(w, "backend: (none)")
	}
	return ExitOK
}

// planBucketEntry is one bucket's JSON projection — its integer
// priority plus the plugin names in topo-sorted order.
type planBucketEntry struct {
	Priority int      `json:"priority"`
	Plugins  []string `json:"plugins"`
}

// writeJSON renders the plan as a single JSON object suitable for
// machine consumption.
func (*PlanCommand) writeJSON(env *Env, plan *pipeline.Plan) int {
	type planJSON struct {
		Frontends  []string          `json:"frontends"`
		Annotators []planBucketEntry `json:"annotators"`
		Generators []planBucketEntry `json:"generators"`
		Backend    string            `json:"backend,omitempty"`
	}
	out := planJSON{
		Frontends:  pluginNames(plan.Frontends),
		Annotators: annotatorBucketsJSON(plan.AnnotatorBuckets),
		Generators: generatorBucketsJSON(plan.GeneratorBuckets),
	}
	if plan.Backend != nil {
		out.Backend = plan.Backend.Name()
	}
	if err := encodeJSONLine(env.Stdout, out); err != nil {
		writeErr(env, "%v", err)
		return ExitInternalError
	}
	return ExitOK
}

// pluginNames returns Name() for each plugin in the slice. Used
// to flatten frontend / backend lists for JSON output.
func pluginNames[T plugin.Plugin](plugins []T) []string {
	out := make([]string, 0, len(plugins))
	for _, p := range plugins {
		out = append(out, p.Name())
	}
	return out
}

// annotatorBucketsJSON / generatorBucketsJSON project the
// per-bucket plugin lists into [planBucketEntry] values. Bucket
// priority renders as the raw integer; readers map back to the
// [priority.AnnotatorShape] / [priority.GeneratorFoundation]
// constants when they need names.
func annotatorBucketsJSON(buckets []pipeline.AnnotatorBucket) []planBucketEntry {
	out := make([]planBucketEntry, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, planBucketEntry{
			Priority: int(b.Priority),
			Plugins:  pluginNames(b.Plugins),
		})
	}
	return out
}

func generatorBucketsJSON(buckets []pipeline.GeneratorBucket) []planBucketEntry {
	out := make([]planBucketEntry, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, planBucketEntry{
			Priority: int(b.Priority),
			Plugins:  pluginNames(b.Plugins),
		})
	}
	return out
}
