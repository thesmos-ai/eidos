// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// Builder accumulates plugins, sinks, cache, and run options. Call
// [Builder.Build] to validate the configuration and return the
// runnable [Pipeline]; chained With* methods return the receiver
// so registrations read top-to-bottom in source order.
//
// The zero value is unusable; construct via [New].
type Builder struct {
	frontends       []plugin.Frontend
	annotators      []plugin.Annotator
	generators      []plugin.Generator
	backends        []plugin.Backend
	directives      []directive.Schema
	directivePrefix string
	sink            sink.Sink
	cache           cache.Cache
	diag            *diag.Sink
	verbose         bool
	parallel        map[Phase]bool
	manifestPath    string
	options         map[string]map[string]string
}

// New returns an empty Builder ready to accept plugins.
func New() *Builder {
	return &Builder{
		options:  map[string]map[string]string{},
		parallel: map[Phase]bool{},
	}
}

// WithParallel opts the listed phases into within-bucket parallel
// execution. Defaults to sequential for every phase when the option
// is omitted. See [Phase] for per-phase semantics and constraints
// (annotators with disjoint Provides; generators that opt in via
// [plugin.NodesOnly]).
func (b *Builder) WithParallel(phases ...Phase) *Builder {
	for _, p := range phases {
		b.parallel[p] = true
	}
	return b
}

// WithManifestPath configures the path the pipeline writes its
// run-end manifest to. The manifest is the per-run record of every
// output file the backend produced, attributed to its plugin and
// hashed. Empty path (the default) disables manifest writing.
func (b *Builder) WithManifestPath(path string) *Builder {
	b.manifestPath = path
	return b
}

// WithDirective registers one or more [directive.Schema] values
// the pipeline's frontends will validate parsed directives against.
// Schemas are shared contracts — multiple plugins may consume the
// same directive — so they are not owned by any plugin and instead
// live on the pipeline's [directive.Registry].
//
// Build rejects two schemas registering under the same name with
// [ErrDuplicateDirective]. Repeat WithDirective calls accumulate;
// schemas may be passed individually or in batch from a package's
// exported list.
func (b *Builder) WithDirective(schemas ...directive.Schema) *Builder {
	b.directives = append(b.directives, schemas...)
	return b
}

// WithDirectivePrefix overrides the directive prefix the pipeline's
// parser recognises. Defaults to [directive.DefaultPrefix] ("gen"),
// matching the project-wide convention; consumers building a custom
// CLI on top of eidos may set a project-specific prefix (e.g.
// "myproject") here so their generated comments don't collide with
// other tools.
//
// The empty string is rejected at Build time as
// [directive.ErrInvalidPrefix].
func (b *Builder) WithDirectivePrefix(prefix string) *Builder {
	b.directivePrefix = prefix
	return b
}

// WithFrontend registers p as a frontend. Multiple frontends may
// coexist; each runs in the frontend phase and writes to the shared
// store.
func (b *Builder) WithFrontend(p plugin.Frontend) *Builder {
	b.frontends = append(b.frontends, p)
	return b
}

// WithAnnotator registers p as an annotator. Annotators run in the
// annotator phase, ordered by priority bucket and capability topo
// (resolved at Build).
func (b *Builder) WithAnnotator(p plugin.Annotator) *Builder {
	b.annotators = append(b.annotators, p)
	return b
}

// WithGenerator registers p as a generator. Generators run in the
// generator phase, ordered by priority bucket and capability topo.
func (b *Builder) WithGenerator(p plugin.Generator) *Builder {
	b.generators = append(b.generators, p)
	return b
}

// WithBackend registers p as the pipeline's backend. Build rejects
// configurations with zero or more than one backend.
func (b *Builder) WithBackend(p plugin.Backend) *Builder {
	b.backends = append(b.backends, p)
	return b
}

// WithSink configures the destination sink the backend writes
// through. Repeated calls replace the previous sink; combine with
// [sink.NewMulti] to fan out to multiple sinks.
func (b *Builder) WithSink(s sink.Sink) *Builder {
	b.sink = s
	return b
}

// WithCache configures the [cache.Cache] used to memoise plugin
// outputs. Defaults to [cache.None] when not set.
func (b *Builder) WithCache(c cache.Cache) *Builder {
	b.cache = c
	return b
}

// WithDiag configures the diagnostic sink Build and Run write
// diagnostics to. Defaults to a fresh [diag.New] when not set.
func (b *Builder) WithDiag(s *diag.Sink) *Builder {
	b.diag = s
	return b
}

// WithVerbose enables verbose-mode diagnostics for the run.
func (b *Builder) WithVerbose(v bool) *Builder {
	b.verbose = v
	return b
}

// WithPluginOptions supplies the named plugin's configuration as a
// string-keyed map. Build calls [plugin.OptionsProvider.SetOptions]
// on plugins that implement the capability and surfaces validation
// errors as diagnostics.
//
// Subsequent calls for the same plugin name replace the prior map.
func (b *Builder) WithPluginOptions(name string, kv map[string]string) *Builder {
	dup := make(map[string]string, len(kv))
	maps.Copy(dup, kv)
	b.options[name] = dup
	return b
}

// Build validates the registered plugins and configuration and
// returns the runnable [Pipeline]. On validation failure, Build
// writes one diagnostic per problem to the configured (or default)
// [diag.Sink] and returns the joined error.
//
// Validation rules at this layer:
//
//   - Plugin names must be unique across every role.
//   - At least one frontend must be registered.
//   - Exactly one backend must be registered.
//   - Plugins implementing [plugin.OptionsProvider] receive their
//     configured options through SetOptions; any returned error
//     surfaces as [ErrInvalidOptions] wrapped with the plugin name
//     and underlying validation cause.
//
// Plan resolution runs after the structural checks pass:
//
//   - Annotators and generators are grouped into priority buckets
//     and topo-sorted within each bucket using
//     [plugin.CapabilityProvider]; cycles surface as [ErrCycle].
//   - Two plugins claiming the same Provides name in the same
//     bucket surface as [ErrDuplicateProvider].
//   - Two plugins claiming the same [plugin.TemplateProvider]
//     func name for the backend's language surface as
//     [ErrTemplateFuncCollision].
func (b *Builder) Build() (*Pipeline, error) {
	if b.diag == nil {
		b.diag = diag.New()
	}
	if b.cache == nil {
		b.cache = cache.NewNone()
	}

	dups := b.validateNoDuplicateNames()
	roles := b.validateRoleCounts()
	options := b.applyAndValidateOptions()
	structural := make([]error, 0, len(dups)+len(roles)+len(options))
	structural = append(structural, dups...)
	structural = append(structural, roles...)
	structural = append(structural, options...)

	if len(structural) > 0 {
		b.emitErrors(structural)
		return nil, errors.Join(structural...)
	}

	plan, planErrs := b.resolvePlan()
	registry, regErrs := b.buildDirectiveRegistry()
	parser, parserErr := b.buildDirectiveParser()
	versionErrs := b.validateEmitVersions()
	postStructural := make([]error, 0, len(planErrs)+len(regErrs)+len(versionErrs)+1)
	postStructural = append(postStructural, planErrs...)
	postStructural = append(postStructural, regErrs...)
	if parserErr != nil {
		postStructural = append(postStructural, parserErr)
	}
	postStructural = append(postStructural, versionErrs...)
	if len(postStructural) > 0 {
		b.emitErrors(postStructural)
		return nil, errors.Join(postStructural...)
	}

	return &Pipeline{
		frontends:    b.frontends,
		annotators:   b.annotators,
		generators:   b.generators,
		backend:      b.backends[0],
		sink:         b.sink,
		cache:        b.cache,
		diag:         b.diag,
		verbose:      b.verbose,
		parallel:     b.parallel,
		manifestPath: b.manifestPath,
		plan:         plan,
		registry:     registry,
		parser:       parser,
	}, nil
}

// buildDirectiveParser constructs the [directive.Parser] the
// pipeline's frontends use to recognise `<prefix>:NAME …`
// directives. The prefix defaults to [directive.DefaultPrefix]
// ("gen") and is overridden via [Builder.WithDirectivePrefix];
// invalid prefixes surface as the wrapped construction error so
// Build emits a positioned diagnostic and refuses to construct a
// half-configured pipeline.
func (b *Builder) buildDirectiveParser() (*directive.Parser, error) {
	prefix := b.directivePrefix
	if prefix == "" {
		prefix = directive.DefaultPrefix
	}
	p, err := directive.NewParser(prefix)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDirectivePrefix, err)
	}
	return p, nil
}

// validateEmitVersions returns one error per plugin whose declared
// [plugin.EmitVersioned] support list does not include the in-tree
// [emit.Major]. Plugins that don't implement EmitVersioned are
// assumed compatible.
func (b *Builder) validateEmitVersions() []error {
	want := emit.Major()
	var errs []error
	check := func(p plugin.Plugin) {
		ev, ok := any(p).(plugin.EmitVersioned)
		if !ok {
			return
		}
		declared := ev.EmitVersions()
		if slices.Contains(declared, want) {
			return
		}
		errs = append(errs, fmt.Errorf("%w: %s declares %v; current emit major is %q",
			ErrIncompatibleEmitVersion, p.Name(), declared, want))
	}
	for _, p := range b.frontends {
		check(p)
	}
	for _, p := range b.annotators {
		check(p)
	}
	for _, p := range b.generators {
		check(p)
	}
	for _, p := range b.backends {
		check(p)
	}
	return errs
}

// buildDirectiveRegistry constructs a [directive.Registry] populated
// from every schema supplied via [Builder.WithDirective] plus every
// schema returned by plugins that implement
// [plugin.DirectiveProvider]. Schema name conflicts surface as
// [ErrDuplicateDirective] (wrapping the underlying
// [directive.ErrSchemaConflict]); the returned registry is non-nil
// even when errors occur so the caller can still expose the partial
// registry for diagnostics.
//
// Auto-collection from DirectiveProvider plugins runs after the
// builder-supplied schemas so manual WithDirective calls win on
// collision — the surface is deliberately tilted toward the
// caller's explicit declarations.
func (b *Builder) buildDirectiveRegistry() (*directive.Registry, []error) {
	r := directive.NewRegistry()
	var errs []error
	for _, s := range b.directives {
		if err := r.Register(s); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrDuplicateDirective, err))
		}
	}
	for _, p := range allPlugins(b.frontends, b.annotators, b.generators, b.backends) {
		dp, ok := p.(plugin.DirectiveProvider)
		if !ok {
			continue
		}
		for _, s := range dp.Directives() {
			if err := r.Register(s); err != nil {
				errs = append(errs, fmt.Errorf("%w: %w", ErrDuplicateDirective, err))
			}
		}
	}
	return r, errs
}

// emitErrors writes one diagnostic per supplied error to the
// builder's [diag.Sink] under the "pipeline" attribution.
func (b *Builder) emitErrors(errs []error) {
	ps := b.diag.For("pipeline")
	for _, e := range errs {
		ps.Errorf(position.Pos{}, "%s", e.Error())
	}
}

// resolvePlan computes the execution-ordered [Plan] from the
// validated registrations: annotators and generators by priority +
// topo, frontends in registration order, the single backend
// directly. Returns a non-empty error slice when bucket resolution
// or template-func validation fails.
func (b *Builder) resolvePlan() (*Plan, []error) {
	annotators, annBuckets, annErr := resolvePhase(b.annotators)
	generators, genBuckets, genErr := resolvePhase(b.generators)

	asPlugins := allPlugins(b.frontends, b.annotators, b.generators, b.backends)
	tplErrs := validateTemplateFuncs(asPlugins, b.backends[0].Language())

	errs := make([]error, 0, 2+len(tplErrs))
	if annErr != nil {
		errs = append(errs, annErr)
	}
	if genErr != nil {
		errs = append(errs, genErr)
	}
	errs = append(errs, tplErrs...)

	if len(errs) > 0 {
		return nil, errs
	}

	annTyped := make([]AnnotatorBucket, len(annBuckets))
	for i, x := range annBuckets {
		annTyped[i] = AnnotatorBucket{Priority: x.Priority, Plugins: x.Plugins}
	}
	genTyped := make([]GeneratorBucket, len(genBuckets))
	for i, x := range genBuckets {
		genTyped[i] = GeneratorBucket{Priority: x.Priority, Plugins: x.Plugins}
	}

	return &Plan{
		Frontends:        b.frontends,
		Annotators:       annotators,
		AnnotatorBuckets: annTyped,
		Generators:       generators,
		GeneratorBuckets: genTyped,
		Backend:          b.backends[0],
	}, nil
}

// allPlugins returns the supplied role slices flattened into a
// single []plugin.Plugin, preserving registration order across
// roles. Used by template-func validation which is role-agnostic.
func allPlugins(
	frontends []plugin.Frontend,
	annotators []plugin.Annotator,
	generators []plugin.Generator,
	backends []plugin.Backend,
) []plugin.Plugin {
	out := make([]plugin.Plugin, 0, len(frontends)+len(annotators)+len(generators)+len(backends))
	for _, p := range frontends {
		out = append(out, p)
	}
	for _, p := range annotators {
		out = append(out, p)
	}
	for _, p := range generators {
		out = append(out, p)
	}
	for _, p := range backends {
		out = append(out, p)
	}
	return out
}

// validateNoDuplicateNames returns one error per duplicate plugin
// name observed across every registered role.
func (b *Builder) validateNoDuplicateNames() []error {
	seen := map[string]struct{}{}
	var errs []error
	check := func(name string) {
		if name == "" {
			return
		}
		if _, dup := seen[name]; dup {
			errs = append(errs, fmt.Errorf("%w: %q", ErrDuplicatePlugin, name))
			return
		}
		seen[name] = struct{}{}
	}
	for _, p := range b.frontends {
		check(p.Name())
	}
	for _, p := range b.annotators {
		check(p.Name())
	}
	for _, p := range b.generators {
		check(p.Name())
	}
	for _, p := range b.backends {
		check(p.Name())
	}
	return errs
}

// validateRoleCounts returns errors when the frontend / backend
// cardinality requirements are violated.
func (b *Builder) validateRoleCounts() []error {
	var errs []error
	if len(b.frontends) == 0 {
		errs = append(errs, ErrNoFrontend)
	}
	switch len(b.backends) {
	case 0:
		errs = append(errs, ErrNoBackend)
	case 1:
		// expected
	default:
		errs = append(errs, ErrMultipleBackends)
	}
	return errs
}

// applyAndValidateOptions calls [plugin.OptionsProvider.SetOptions]
// on every plugin that implements the capability. Validation errors
// surface as [ErrInvalidOptions] wrapped with the plugin name.
func (b *Builder) applyAndValidateOptions() []error {
	var errs []error
	apply := func(p plugin.Plugin) {
		op, ok := p.(plugin.OptionsProvider)
		if !ok {
			return
		}
		schema := op.OptionsSchema()
		options := opt.New(schema, b.options[p.Name()])
		if err := op.SetOptions(options); err != nil {
			errs = append(errs, fmt.Errorf("%w: %s: %w", ErrInvalidOptions, p.Name(), err))
		}
	}
	for _, p := range b.frontends {
		apply(p)
	}
	for _, p := range b.annotators {
		apply(p)
	}
	for _, p := range b.generators {
		apply(p)
	}
	for _, p := range b.backends {
		apply(p)
	}
	return errs
}
