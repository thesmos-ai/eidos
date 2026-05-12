// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// Run executes the resolved [Plan] against a fresh [store.Store],
// in plan order: every frontend, then every annotator, then every
// generator, then the backend. Each plugin gets its own
// [store.Reader] so read-tracking is per-plugin (the recorded reads
// later feed the cache layer).
//
// Run is run-to-completion: a non-nil error from any plugin's role
// method becomes a [diag.Error] diagnostic attributed to the plugin
// and execution continues with the next plugin in the same phase
// (and the next phase). Plugin code that panics is contained by a
// [diag.RecoverAs] guard installed at the plugin invocation
// boundary; the panic surfaces as an [diag.Error] with a stack
// trace and the next plugin still runs. Plugin code that emits
// diagnostics directly to ctx.Diag is captured the same way.
//
// After every plugin has run, Run returns [ErrRunHadErrors] when
// any [diag.Error] diagnostic was recorded; otherwise nil. Call
// [Pipeline.Diag] to inspect the per-error details.
//
// Returns [ErrNoSink] without running any phase when no
// [sink.Sink] was configured at Build time — the backend has
// nowhere to write so the run cannot meaningfully complete.
//
// patterns is the per-frontend input list (typically Go-style
// import paths or filesystem globs). Each frontend receives every
// pattern. When [Builder.WithVerbose] was set the pipeline emits
// per-phase Info diagnostics so the user can see progress without
// turning on per-plugin tracing.
func (p *Pipeline) Run(ctx context.Context, patterns ...string) error {
	if p.sink == nil {
		return ErrNoSink
	}
	_ = ctx // reserved for cancellation in a later milestone

	// Wrap the configured sink with a recording wrapper so the
	// pipeline can compose a manifest from every captured write at
	// run end. The wrapper writes through to the inner sink so
	// backend output still reaches its destination.
	recorder := newRecordingSink(p.sink)

	s := store.New()
	p.lastStore.Store(s)
	p.runFrontends(s, patterns)
	s.Nodes().Freeze() // post-frontend: node structure is frozen
	p.runAnnotators(s)
	p.runDirectiveOverride(s)
	p.runGenerators(s)
	p.runLayout(s)    // compose Target on every emit entity before structure freeze
	s.Emit().Freeze() // post-generator + post-layout: emit structure is frozen
	p.runBackend(s, recorder)
	p.writeManifest(recorder, s)
	p.logRunSummary()

	if p.diag.HasErrors() {
		return ErrRunHadErrors
	}
	return nil
}

// writeManifest writes the run-end manifest when a path is
// configured via [Builder.WithManifestPath]. Manifest write errors
// surface as Warn diagnostics (the manifest is observability, not
// correctness) so a manifest-write failure does not turn the run
// into a failed one.
func (p *Pipeline) writeManifest(rec *recordingSink, s *store.Store) {
	if p.manifestPath == "" {
		return
	}
	m := rec.asManifest(time.Now().UTC().Format(time.RFC3339), s, p.pluginNames())
	if err := manifest.Write(p.manifestPath, m); err != nil {
		p.diag.For("pipeline").Warnf(position.Pos{}, "manifest write failed: %v", err)
	}
}

// pluginNames returns the registered plugins' [plugin.Plugin.Name]
// values in registration order — frontends, annotators, generators,
// then the backend. The manifest's per-output Plugins list quotes
// this slice so every entry shares the run's plugin universe; the
// rendered file's `Plugins:` header is composed from the same set,
// so manifest and on-disk provenance stay aligned.
func (p *Pipeline) pluginNames() []string {
	out := make([]string, 0, len(p.frontends)+len(p.annotators)+len(p.generators)+1)
	for _, fe := range p.frontends {
		out = append(out, fe.Name())
	}
	for _, ann := range p.annotators {
		out = append(out, ann.Name())
	}
	for _, gen := range p.generators {
		out = append(out, gen.Name())
	}
	if p.backend != nil {
		out = append(out, p.backend.Name())
	}
	return out
}

// DryRun returns the resolved [Plan] without executing any phase.
// Tooling such as "eidos explain plan" calls DryRun to display the
// resolved order and any Build-time diagnostics without writing
// files. The supplied context is reserved for cancellation in a
// later milestone.
func (p *Pipeline) DryRun(ctx context.Context) *Plan {
	_ = ctx
	return p.plan
}

// runFrontends invokes Load on every frontend for every pattern.
// Per-call errors and panics become Error diagnostics attributed to
// the frontend's name; subsequent frontends and patterns still run.
// When [PhaseFrontend] is opted into via [Builder.WithParallel] the
// frontend×pattern invocations dispatch concurrently.
func (p *Pipeline) runFrontends(s *store.Store, patterns []string) {
	p.logPhaseStart("frontend", "%d frontend(s), %d pattern(s)", len(p.plan.Frontends), len(patterns))
	if p.parallel[PhaseFrontend] {
		var wg sync.WaitGroup
		for _, fe := range p.plan.Frontends {
			for _, pattern := range patterns {
				wg.Go(func() { p.invokeFrontend(fe, pattern, s) })
			}
		}
		wg.Wait()
		return
	}
	for _, fe := range p.plan.Frontends {
		for _, pattern := range patterns {
			p.invokeFrontend(fe, pattern, s)
		}
	}
}

// invokeFrontend runs one Frontend.Load call with panic containment
// so a misbehaving frontend cannot abort the entire run.
func (p *Pipeline) invokeFrontend(fe plugin.Frontend, pattern string, s *store.Store) {
	ps := p.diag.For(fe.Name())
	defer diag.RecoverAs(ps, position.Pos{})
	ctx := &plugin.FrontendContext{
		Store:    s,
		Diag:     p.diag,
		Registry: p.registry,
		Parser:   p.parser,
		Cache:    p.cache,
		Pattern:  pattern,
	}
	if err := fe.Load(ctx); err != nil {
		p.reportPluginError(ps, fe.Name(), fmt.Sprintf("frontend Load(%q)", pattern), err)
	}
}

// reportPluginError records err as a diagnostic attributed to a
// specific plugin. Errors wrapping [store.ErrFrozen] indicate a
// framework-contract violation (a plugin mutated a view it should
// not have touched) and surface at Internal severity so operators
// can distinguish them from ordinary user-side problems. Every
// other error becomes a normal Error diagnostic.
func (p *Pipeline) reportPluginError(ps *diag.PluginSink, name, role string, err error) {
	if errors.Is(err, store.ErrFrozen) {
		p.diag.Internalf(position.Pos{}, "%s %q violated frozen-store contract: %v", role, name, err)
		return
	}
	ps.Errorf(position.Pos{}, "%s failed: %v", role, err)
}

// runAnnotators invokes Annotate on every annotator. Buckets run
// in ascending priority order; within a bucket plugins run in
// topo-sorted order sequentially, OR concurrently when
// [PhaseAnnotator] is enabled via [Builder.WithParallel] AND every
// plugin in the bucket has pairwise-disjoint [plugin.CapabilityProvider.Provides]
// (per spec §18). Buckets that fail the disjoint check fall back to
// sequential to preserve write-order semantics.
func (p *Pipeline) runAnnotators(s *store.Store) {
	p.logPhaseStart("annotator", "%d annotator(s)", len(p.plan.Annotators))
	for _, bucket := range p.plan.AnnotatorBuckets {
		if p.parallel[PhaseAnnotator] {
			// Build rejects same-bucket plugins that claim the same
			// Provides name (ErrDuplicateProvider), so by the time
			// the runtime sees a bucket every plugin's Provides
			// set is pairwise disjoint and the bucket may safely
			// dispatch concurrently.
			var wg sync.WaitGroup
			for _, ann := range bucket.Plugins {
				wg.Go(func() { p.invokeAnnotator(ann, s) })
			}
			wg.Wait()
			continue
		}
		for _, ann := range bucket.Plugins {
			p.invokeAnnotator(ann, s)
		}
	}
}

// invokeAnnotator runs one Annotator.Annotate call with panic
// containment. After the call returns the recorded
// [store.ReadSet.Hash] is written to the cache under a
// per-plugin key so cache-aware downstream tooling can detect
// "this plugin ran with these reads".
func (p *Pipeline) invokeAnnotator(ann plugin.Annotator, s *store.Store) {
	ps := p.diag.For(ann.Name())
	defer diag.RecoverAs(ps, position.Pos{})
	r := p.newReader(s)
	ctx := &plugin.AnnotatorContext{
		Store:  s,
		Reader: r,
		Diag:   p.diag,
	}
	if err := ann.Annotate(ctx); err != nil {
		p.reportPluginError(ps, ann.Name(), "annotator", err)
	}
	p.recordCacheKey(ann.Name(), r)
}

// runGenerators invokes Generate on every generator. Buckets run
// in ascending priority order; within a bucket plugins run in
// topo-sorted order sequentially, OR concurrently when
// [PhaseGenerator] is enabled via [Builder.WithParallel] AND every
// plugin in the bucket implements [plugin.NodesOnly] returning
// true (i.e. they promise not to read upstream emit). Buckets that
// fail the NodesOnly check fall back to sequential.
func (p *Pipeline) runGenerators(s *store.Store) {
	p.logPhaseStart("generator", "%d generator(s)", len(p.plan.Generators))
	for _, bucket := range p.plan.GeneratorBuckets {
		if p.parallel[PhaseGenerator] && allNodesOnly(bucket.Plugins) {
			var wg sync.WaitGroup
			for _, gen := range bucket.Plugins {
				wg.Go(func() { p.invokeGenerator(gen, s) })
			}
			wg.Wait()
			continue
		}
		for _, gen := range bucket.Plugins {
			p.invokeGenerator(gen, s)
		}
	}
}

// allNodesOnly reports whether every generator in plugins
// implements [plugin.NodesOnly] returning true. A single false /
// non-implementing generator disqualifies the bucket from parallel
// execution because it might read the emit graph another generator
// is mutating.
func allNodesOnly(plugins []plugin.Generator) bool {
	for _, g := range plugins {
		no, ok := any(g).(plugin.NodesOnly)
		if !ok || !no.NodesOnly() {
			return false
		}
	}
	return true
}

// invokeGenerator runs one Generator.Generate call with panic
// containment. After the call returns the recorded
// [store.ReadSet.Hash] is written to the cache under a
// per-plugin key — see [Pipeline.recordCacheKey].
func (p *Pipeline) invokeGenerator(gen plugin.Generator, s *store.Store) {
	ps := p.diag.For(gen.Name())
	defer diag.RecoverAs(ps, position.Pos{})
	r := p.newReader(s)
	ctx := &plugin.GeneratorContext{
		Store:  s,
		Reader: r,
		Diag:   p.diag,
	}
	if err := gen.Generate(ctx); err != nil {
		p.reportPluginError(ps, gen.Name(), "generator", err)
	}
	p.recordCacheKey(gen.Name(), r)
}

// recordCacheKey writes the per-plugin cache marker — a fixed-form
// key derived from the plugin's name and the supplied [store.ReadSet]
// hash — to the configured cache. Cache layers that consume the
// marker can later detect "this plugin ran with the same inputs"
// for skip-on-hit optimisations.
//
// Errors from the cache are silently dropped because the cache is
// best-effort: a failed write is no worse than running without a
// cache at all.
func (p *Pipeline) recordCacheKey(name string, r *store.Reader) {
	key := fmt.Sprintf("plugin:%s:reads:%s", name, r.ReadSet().Hash())
	_ = p.cache.Put(key, []byte(r.ReadSet().Hash())) //nolint:errcheck // best-effort cache marker
}

// runBackend invokes Render on the backend with a populated
// [plugin.BackendContext] including the registered-order plugin
// list (for template-collection enumeration) and the plan-execution
// order (for deterministic override application). Wraps the call
// in a [diag.RecoverAs] guard so a backend panic is contained.
func (p *Pipeline) runBackend(s *store.Store, dst sink.Sink) {
	p.logPhaseStart("backend", "lang=%s", p.backend.Language())
	ps := p.diag.For(p.backend.Name())
	defer diag.RecoverAs(ps, position.Pos{})
	r := p.newReader(s)
	ctx := &plugin.BackendContext{
		Store:      s,
		Reader:     r,
		Diag:       p.diag,
		Sink:       dst,
		Lang:       p.backend.Language(),
		Plugins:    p.registeredPlugins(),
		Ordered:    p.orderedPlugins(),
		Command:    p.commandHeader(),
		SourceRoot: p.sourceRoot,
	}
	if err := p.backend.Render(ctx); err != nil {
		p.reportPluginError(ps, p.backend.Name(), "backend", err)
	}
	p.recordCacheKey(p.backend.Name(), r)
}

// newReader constructs the per-plugin [store.Reader] every plugin
// phase hands to a plugin. When the pipeline carries a scope
// predicate (set via [Builder.WithTargetSymbol]) the returned
// reader pre-filters node-side range queries to in-scope nodes
// transparently; an unconfigured pipeline returns a vanilla
// unscoped reader.
func (p *Pipeline) newReader(s *store.Store) *store.Reader {
	if p.scope == nil {
		return store.NewReader(s)
	}
	return store.NewScopedReader(s, p.scope)
}

// commandHeader returns the literal string to stamp into the
// "Command:" header line of every rendered file. A caller-supplied
// value through [Builder.WithCommand] wins — letting tests and
// library embedders pin a stable value. Empty falls back to
// [commandLine], which renders `os.Args[1:]` for real CLI runs and
// "(library)" when no positional arguments are present.
func (p *Pipeline) commandHeader() string {
	if p.command != "" {
		return p.command
	}
	return commandLine()
}

// commandLine returns the CLI-style rendering of the current
// process's arguments, used as the [plugin.BackendContext.Command]
// default. When the host has no positional arguments (typically
// library / test invocations), returns "(library)" — a stable
// marker that signals programmatic use without leaking
// test-runner flags into the generated output.
//
// Test-runner invocations populate os.Args with per-machine paths
// (`-test.testlogfile=/var/folders/.../testlog.txt`) — a
// determinism leak the [Builder.WithCommand] override exists to
// neutralise.
func commandLine() string {
	if len(os.Args) <= 1 {
		return "(library)"
	}
	return strings.Join(os.Args[1:], " ")
}

// logPhaseStart writes a verbose-mode Info diagnostic announcing
// the phase boundary. No-op when verbose mode is off so silent runs
// stay silent.
func (p *Pipeline) logPhaseStart(phase, format string, args ...any) {
	if !p.verbose {
		return
	}
	ps := p.diag.For("pipeline")
	ps.Infof(position.Pos{}, "phase=%s "+format, append([]any{phase}, args...)...)
}

// logRunSummary writes a verbose-mode Info diagnostic at run-end
// with a count of diagnostics emitted across all phases. No-op
// when verbose mode is off.
func (p *Pipeline) logRunSummary() {
	if !p.verbose {
		return
	}
	ps := p.diag.For("pipeline")
	ps.Infof(position.Pos{}, "run complete: %d error(s), %d warning(s), %d info",
		p.diag.Count(diag.Error), p.diag.Count(diag.Warn), p.diag.Count(diag.Info))
}

// registeredPlugins returns the full list of plugins in user
// registration order: frontends, then annotators, then generators,
// then the backend. The backend uses this list to find every
// [plugin.TemplateProvider] for template merging.
func (p *Pipeline) registeredPlugins() []plugin.Plugin {
	out := make([]plugin.Plugin, 0,
		len(p.frontends)+len(p.annotators)+len(p.generators)+1)
	for _, x := range p.frontends {
		out = append(out, x)
	}
	for _, x := range p.annotators {
		out = append(out, x)
	}
	for _, x := range p.generators {
		out = append(out, x)
	}
	out = append(out, p.backend)
	return out
}

// orderedPlugins returns the full plugin list in plan-execution
// order: frontends (registration order; the frontend role has no
// priority/topo), then annotators (plan order), then generators
// (plan order), then the backend. The backend uses this list to
// apply [plugin.TemplateProvider.TemplateOverrides] deterministically.
func (p *Pipeline) orderedPlugins() []plugin.Plugin {
	out := make([]plugin.Plugin, 0,
		len(p.plan.Frontends)+len(p.plan.Annotators)+len(p.plan.Generators)+1)
	for _, x := range p.plan.Frontends {
		out = append(out, x)
	}
	for _, x := range p.plan.Annotators {
		out = append(out, x)
	}
	for _, x := range p.plan.Generators {
		out = append(out, x)
	}
	out = append(out, p.plan.Backend)
	return out
}
