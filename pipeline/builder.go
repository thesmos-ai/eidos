// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
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
	command         string
	sourceRoot      string
	outputFilename  string
	outputPackage   string
	outputLayout    string
	outputDir       string
	targetSymbol    string

	// Project-level routing layer — the `output.*` block on
	// [cli.Config.Output]. Populated by [Builder.WithProjectOutput];
	// empty when no project-level config is supplied.
	projectLayout  string
	projectPackage string
	projectDir     string

	// Per-plugin routing overrides — the `plugins[*].output.*`
	// block keyed by plugin name. Populated by repeated
	// [Builder.WithPluginOutput] calls.
	pluginOutputs map[string]layoutOverride
}

// layoutOverride captures one layer's contribution to the routing
// merge: any non-empty field overrides the layer below.
type layoutOverride struct {
	Layout  string
	Package string
	Dir     string
}

// New returns an empty Builder ready to accept plugins.
func New() *Builder {
	return &Builder{
		options:       map[string]map[string]string{},
		parallel:      map[Phase]bool{},
		pluginOutputs: map[string]layoutOverride{},
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

// WithCommand sets the literal string the backend stamps into the
// "Command:" header line of every rendered file. Library and test
// callers set a stable value (e.g. "(library)", a synthetic CLI
// rendering) so the header stays byte-identical across runs and
// across machines. Empty falls back to the pipeline's automatic
// rendering of `os.Args[1:]` — the right answer for real CLI
// invocations but a determinism leak under `go test`, where the
// runner injects absolute paths into the flag set.
func (b *Builder) WithCommand(cmd string) *Builder {
	b.command = cmd
	return b
}

// WithSourceRoot sets the base directory the backend renders source
// paths relative to in the "Source:" header line. Each entity's
// origin file path is stripped of this prefix and normalised to
// forward slashes so the rendered output stays byte-identical
// across machines, OSes, and project-root layouts.
//
// When the option is not set, [Builder.Build] resolves the
// SourceRoot from [os.Getwd] at construction time — the value is
// pinned then, not at render time, so [os.Chdir] mid-run does not
// drift the headers. A Getwd failure (rare) leaves the SourceRoot
// empty and the backend falls through to rendering origin paths
// verbatim. The CLI threads [cli.Env.Workdir]; library embedders
// override when their project root differs from the working
// directory.
func (b *Builder) WithSourceRoot(root string) *Builder {
	b.sourceRoot = root
	return b
}

// WithOutputFilename pins [emit.Target.Filename] for every emitted
// decl in scope. Empty leaves the per-decl default in place — the
// origin source basename combined with the contributing plugin's
// declared filename suffix. Maps to the CLI's `-o` override.
//
// `-o` without a scope filter routes every emitted decl from every
// plugin into one rendered file; the CLI rejects that combination
// at flag-parse time, but library callers using this method
// directly should pair it with [Builder.WithTargetSymbol] or risk
// one-file-one-package violations from the Layout phase.
func (b *Builder) WithOutputFilename(name string) *Builder {
	b.outputFilename = name
	return b
}

// WithOutputPackage pins [emit.Target.Package] for every emitted
// decl in scope. Empty inherits the per-decl default (origin's
// source package name under alongside-source layout, the resolved
// policy's package under centralised). Maps to the CLI's `-p`
// override.
func (b *Builder) WithOutputPackage(pkg string) *Builder {
	b.outputPackage = pkg
	return b
}

// WithOutputLayout overrides the project-default layout for this
// run. Accepts [LayoutAlongsideSource] (the framework default) or
// [LayoutCentralised]. Empty falls back to the configured project
// layout, or the framework default when no project layout is set.
// Maps to the CLI's `-layout` flag.
func (b *Builder) WithOutputLayout(layout string) *Builder {
	b.outputLayout = layout
	return b
}

// WithOutputDir sets the rendered output directory under centralised
// layout. Ignored (with a configuration warning at validation time)
// under alongside-source layout, which derives the directory from
// origin. Maps to the CLI's `-output-dir` flag.
func (b *Builder) WithOutputDir(dir string) *Builder {
	b.outputDir = dir
	return b
}

// WithProjectOutput supplies the project-level routing-layer
// policy — the `output.*` block on `.eidos.yaml`. Each non-empty
// argument upgrades the corresponding field's attribution from
// [manifest.LayerFramework] to [manifest.LayerProject] in the
// merge the Layout phase walks; empty arguments leave the layer
// below unchanged. The triple is identical at every plugin name
// (plugin-specific overrides slot in through
// [Builder.WithPluginOutput]).
func (b *Builder) WithProjectOutput(layout, pkg, dir string) *Builder {
	b.projectLayout = layout
	b.projectPackage = pkg
	b.projectDir = dir
	return b
}

// WithPluginOutput supplies a per-plugin routing override — the
// `plugins[*].output.*` block on `.eidos.yaml`. Each non-empty
// argument upgrades the field's attribution to
// [manifest.LayerPerPlugin] in the merge for plugin `name`;
// empty arguments leave the layer below unchanged. Repeated
// calls for the same name replace the prior entry verbatim.
func (b *Builder) WithPluginOutput(name, layout, pkg, dir string) *Builder {
	b.pluginOutputs[name] = layoutOverride{
		Layout:  layout,
		Package: pkg,
		Dir:     dir,
	}
	return b
}

// WithTargetSymbol narrows the run to source decls whose unqualified
// Name equals symbol. Empty disables the filter — every source decl
// participates. The Layout phase consumes the resulting predicate
// through the [store.Reader] each plugin receives, so plugins
// iterate the scoped view transparently. Maps to the CLI's
// `-target` flag.
func (b *Builder) WithTargetSymbol(symbol string) *Builder {
	b.targetSymbol = symbol
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
	registry, directiveOwners, regErrs := b.buildDirectiveRegistry()
	parser, parserErr := b.buildDirectiveParser()
	versionErrs := b.validateEmitVersions()
	outputErrs := b.validateOutputs()
	postStructural := make([]error, 0, len(planErrs)+len(regErrs)+len(versionErrs)+len(outputErrs)+1)
	postStructural = append(postStructural, planErrs...)
	postStructural = append(postStructural, regErrs...)
	if parserErr != nil {
		postStructural = append(postStructural, parserErr)
	}
	postStructural = append(postStructural, versionErrs...)
	postStructural = append(postStructural, outputErrs...)
	if len(postStructural) > 0 {
		b.emitErrors(postStructural)
		return nil, errors.Join(postStructural...)
	}

	defaultPolicy, pluginPolicies := b.resolveLayoutPolicies()
	return &Pipeline{
		frontends:       b.frontends,
		annotators:      b.annotators,
		generators:      b.generators,
		backend:         b.backends[0],
		sink:            b.sink,
		cache:           b.cache,
		diag:            b.diag,
		verbose:         b.verbose,
		parallel:        b.parallel,
		manifestPath:    b.manifestPath,
		command:         b.command,
		sourceRoot:      b.resolveSourceRoot(),
		defaultPolicy:   defaultPolicy,
		pluginPolicies:  pluginPolicies,
		outFilename:     b.outputFilename,
		scope:           b.resolveScope(),
		targetSym:       b.targetSymbol,
		plan:            plan,
		registry:        registry,
		parser:          parser,
		directiveOwners: directiveOwners,
	}, nil
}

// resolveSourceRoot returns the SourceRoot the constructed Pipeline
// stamps onto every BackendContext. A caller-supplied value wins
// outright. Otherwise the resolver falls back to [os.Getwd] at
// Build time so the value is pinned at construction rather than
// being CWD-coupled at render time — long-running consumers that
// [os.Chdir] mid-run still observe stable headers. A Getwd failure
// (rare) falls through to the empty string; the backend then
// renders origin paths verbatim, preserving the previous behaviour
// rather than surfacing a transient os error.
func (b *Builder) resolveSourceRoot() string {
	if b.sourceRoot != "" {
		return b.sourceRoot
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

// resolveLayoutPolicies pre-computes the per-plugin
// [LayoutPolicy] map the Pipeline serves from
// [Pipeline.LayoutPolicyFor]. Every plugin known to the Builder
// gets an entry that walks the precedence layers — framework
// default, project config, per-plugin override, CLI flags — and
// stamps each field's [manifest.Layer] sibling as a higher-
// priority layer takes effect. Plugins with no per-plugin
// override resolve identically; plugins with an override see
// only the fields the override touches diverge from the project
// merge.
//
// The default policy returned alongside the map covers two
// cases: queries against plugin names the Builder doesn't know
// (defensive lookups by tooling that constructs queries before
// the plugin set is finalised), and the initial composition for
// emit decls attributed to a plugin not yet in the per-plugin
// map.
func (b *Builder) resolveLayoutPolicies() (LayoutPolicy, map[string]LayoutPolicy) {
	base := b.mergedBasePolicy()
	out := make(map[string]LayoutPolicy, len(b.pluginOutputs))
	for _, p := range b.frontends {
		out[p.Name()] = b.applyPerPluginAndCLI(base, p.Name())
	}
	for _, p := range b.annotators {
		out[p.Name()] = b.applyPerPluginAndCLI(base, p.Name())
	}
	for _, p := range b.generators {
		out[p.Name()] = b.applyPerPluginAndCLI(base, p.Name())
	}
	for _, p := range b.backends {
		out[p.Name()] = b.applyPerPluginAndCLI(base, p.Name())
	}
	return b.applyPerPluginAndCLI(base, ""), out
}

// mergedBasePolicy returns the framework-default policy with the
// project-level layer applied on top. The result is the
// pre-per-plugin / pre-CLI merge every plugin starts from.
func (b *Builder) mergedBasePolicy() LayoutPolicy {
	policy := NewLayoutPolicy()
	if b.projectLayout != "" {
		policy.Layout = b.projectLayout
		policy.LayoutFrom = manifest.LayerProject
	}
	if b.projectPackage != "" {
		policy.Package = b.projectPackage
		policy.PackageFrom = manifest.LayerProject
	}
	if b.projectDir != "" {
		policy.Dir = b.projectDir
		policy.DirFrom = manifest.LayerProject
	}
	return policy
}

// applyPerPluginAndCLI applies the per-plugin override (when
// `name` matches a configured entry) and then the CLI flags on
// top of base. An empty name skips the per-plugin layer — used
// to compute the default policy for unknown plugin names.
func (b *Builder) applyPerPluginAndCLI(base LayoutPolicy, name string) LayoutPolicy {
	policy := base
	if name != "" {
		if over, ok := b.pluginOutputs[name]; ok {
			if over.Layout != "" {
				policy.Layout = over.Layout
				policy.LayoutFrom = manifest.LayerPerPlugin
			}
			if over.Package != "" {
				policy.Package = over.Package
				policy.PackageFrom = manifest.LayerPerPlugin
			}
			if over.Dir != "" {
				policy.Dir = over.Dir
				policy.DirFrom = manifest.LayerPerPlugin
			}
		}
	}
	if b.outputLayout != "" {
		policy.Layout = b.outputLayout
		policy.LayoutFrom = manifest.LayerCLI
	}
	if b.outputPackage != "" {
		policy.Package = b.outputPackage
		policy.PackageFrom = manifest.LayerCLI
	}
	if b.outputDir != "" {
		policy.Dir = b.outputDir
		policy.DirFrom = manifest.LayerCLI
	}
	return policy
}

// resolveScope returns the [store.ScopePredicate] the Pipeline
// passes to every per-plugin Reader. An empty target symbol
// returns nil — Readers run unfiltered. A non-empty symbol
// produces a predicate matching every source node whose
// unqualified Name equals symbol, walking method / field
// receivers via their Owner chain so a `-target Article`
// invocation also passes Article's methods and fields.
func (b *Builder) resolveScope() store.ScopePredicate {
	if b.targetSymbol == "" {
		return nil
	}
	return scopeBySymbol(b.targetSymbol)
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

// validateOutputs returns one error per [plugin.FilenameProvider]
// whose Outputs slice for the active backend's language violates
// any of the shape rules the framework enforces: every Suffix
// must be non-empty; tags within the slice must be unique; at
// most one Output may declare an empty Tag; the empty-Tag Output
// must be at index 0 when present. The rules protect every
// downstream consumer (Layout dispatch, directive resolution,
// CLI scoping, manifest attribution) from ambiguous routing
// states — a malformed slice would silently collapse two outputs
// into one filename or leave a tagged decl with no matching
// suffix to resolve against.
//
// Plugins returning a nil or empty slice for the active language
// are not validated by this hook — they signal "no routable
// output" and surface
// [ErrMissingFilenameProvider] from the Layout phase only if
// they actually emit a routable decl.
//
// The Build-time check uses the backend's language even though
// plugins may ship outputs for languages the configured backend
// does not target — only the active language matters at run
// time. Validation against unused languages would be both
// pointless (the framework never reads those entries) and
// surprising (a plugin shipping malformed entries for an
// unsupported language would fail Build despite the run never
// touching those entries).
func (b *Builder) validateOutputs() []error {
	if len(b.backends) == 0 {
		return nil
	}
	lang := b.backends[0].Language()
	var errs []error
	for _, gen := range b.generators {
		fp, ok := any(gen).(plugin.FilenameProvider)
		if !ok {
			continue
		}
		outputs := fp.Outputs(lang)
		if len(outputs) == 0 {
			continue
		}
		seenTags := make(map[string]int, len(outputs))
		emptyTagCount := 0
		for i, o := range outputs {
			if o.Suffix == "" {
				errs = append(errs, fmt.Errorf(
					"%w: %s: outputs[%d]: Suffix is required",
					ErrInvalidOutputs, gen.Name(), i,
				))
			}
			if prev, dup := seenTags[o.Tag]; dup {
				errs = append(errs, fmt.Errorf(
					"%w: %s: outputs declare tag %q at indices %d and %d",
					ErrInvalidOutputs, gen.Name(), o.Tag, prev, i,
				))
			}
			seenTags[o.Tag] = i
			if o.Tag == "" {
				emptyTagCount++
				if i != 0 {
					errs = append(errs, fmt.Errorf(
						"%w: %s: outputs[%d]: empty-Tag output must be declared at index 0",
						ErrInvalidOutputs, gen.Name(), i,
					))
				}
			}
		}
		if emptyTagCount > 1 {
			errs = append(errs, fmt.Errorf(
				"%w: %s: %d outputs declare an empty Tag; at most one is permitted (the plugin's primary output)",
				ErrInvalidOutputs, gen.Name(), emptyTagCount,
			))
		}
	}
	return errs
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
func (b *Builder) buildDirectiveRegistry() (*directive.Registry, map[directive.Name]string, []error) {
	r := directive.NewRegistry()
	owners := map[directive.Name]string{}
	var errs []error
	// Register the framework's core directives first so they're
	// reserved against accidental override by builder-supplied or
	// plugin-supplied schemas — a downstream attempt to redefine
	// `out` would surface as ErrDuplicateDirective.
	for _, s := range coreDirectives() {
		// coreDirectives() returns the framework's reserved schemas
		// into a fresh registry — registration cannot collide here,
		// so the error from Register is discarded.
		_ = r.Register(s) //nolint:errcheck // unreachable: fresh registry + unique core schema names
	}
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
				continue
			}
			// Record the plugin that owns this directive so the
			// Layout phase can resolve per-directive `out=` / `pkg=`
			// keys against the matching plugin's output.
			owners[s.Name] = p.Name()
		}
	}
	return r, owners, errs
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
