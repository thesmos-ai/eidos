// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"context"
	"errors"
	"fmt"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/plugin"
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

	s := store.New()
	p.runFrontends(s, patterns)
	s.Nodes().Freeze() // post-frontend: node structure is frozen
	p.runAnnotators(s)
	p.runDirectiveOverride(s)
	p.runGenerators(s)
	s.Emit().Freeze() // post-generator: emit structure is frozen
	p.runBackend(s)
	p.logRunSummary()

	if p.diag.HasErrors() {
		return ErrRunHadErrors
	}
	return nil
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
func (p *Pipeline) runFrontends(s *store.Store, patterns []string) {
	p.logPhaseStart("frontend", "%d frontend(s), %d pattern(s)", len(p.plan.Frontends), len(patterns))
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
	if err := fe.Load(pattern, s, p.diag); err != nil {
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

// runAnnotators invokes Annotate on every annotator in plan order.
// Each annotator gets its own [store.Reader] for read-tracking.
func (p *Pipeline) runAnnotators(s *store.Store) {
	p.logPhaseStart("annotator", "%d annotator(s)", len(p.plan.Annotators))
	for _, ann := range p.plan.Annotators {
		p.invokeAnnotator(ann, s)
	}
}

// invokeAnnotator runs one Annotator.Annotate call with panic
// containment.
func (p *Pipeline) invokeAnnotator(ann plugin.Annotator, s *store.Store) {
	ps := p.diag.For(ann.Name())
	defer diag.RecoverAs(ps, position.Pos{})
	ctx := &plugin.AnnotatorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   p.diag,
	}
	if err := ann.Annotate(ctx); err != nil {
		p.reportPluginError(ps, ann.Name(), "annotator", err)
	}
}

// runGenerators invokes Generate on every generator in plan order.
func (p *Pipeline) runGenerators(s *store.Store) {
	p.logPhaseStart("generator", "%d generator(s)", len(p.plan.Generators))
	for _, gen := range p.plan.Generators {
		p.invokeGenerator(gen, s)
	}
}

// invokeGenerator runs one Generator.Generate call with panic
// containment.
func (p *Pipeline) invokeGenerator(gen plugin.Generator, s *store.Store) {
	ps := p.diag.For(gen.Name())
	defer diag.RecoverAs(ps, position.Pos{})
	ctx := &plugin.GeneratorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   p.diag,
	}
	if err := gen.Generate(ctx); err != nil {
		p.reportPluginError(ps, gen.Name(), "generator", err)
	}
}

// runBackend invokes Render on the backend with a populated
// [plugin.BackendContext] including the registered-order plugin
// list (for template-collection enumeration) and the plan-execution
// order (for deterministic override application). Wraps the call
// in a [diag.RecoverAs] guard so a backend panic is contained.
func (p *Pipeline) runBackend(s *store.Store) {
	p.logPhaseStart("backend", "lang=%s", p.backend.Language())
	ps := p.diag.For(p.backend.Name())
	defer diag.RecoverAs(ps, position.Pos{})
	ctx := &plugin.BackendContext{
		Store:   s,
		Reader:  store.NewReader(s),
		Diag:    p.diag,
		Sink:    p.sink,
		Lang:    p.backend.Language(),
		Plugins: p.registeredPlugins(),
		Ordered: p.orderedPlugins(),
	}
	if err := p.backend.Render(ctx); err != nil {
		p.reportPluginError(ps, p.backend.Name(), "backend", err)
	}
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
