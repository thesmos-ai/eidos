// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"errors"
	"fmt"
	"maps"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/core/position"
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
	frontends  []plugin.Frontend
	annotators []plugin.Annotator
	generators []plugin.Generator
	backends   []plugin.Backend
	sink       sink.Sink
	cache      cache.Cache
	diag       *diag.Sink
	verbose    bool
	options    map[string]map[string]string
}

// New returns an empty Builder ready to accept plugins.
func New() *Builder {
	return &Builder{options: map[string]map[string]string{}}
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
// Cycles in [plugin.CapabilityProvider.Requires], duplicate
// template func extension names, and plan resolution land in a
// later milestone.
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
	errs := make([]error, 0, len(dups)+len(roles)+len(options))
	errs = append(errs, dups...)
	errs = append(errs, roles...)
	errs = append(errs, options...)

	if len(errs) > 0 {
		ps := b.diag.For("pipeline")
		for _, e := range errs {
			ps.Errorf(position.Pos{}, "%s", e.Error())
		}
		return nil, errors.Join(errs...)
	}

	return &Pipeline{
		frontends:  b.frontends,
		annotators: b.annotators,
		generators: b.generators,
		backend:    b.backends[0],
		sink:       b.sink,
		cache:      b.cache,
		diag:       b.diag,
		verbose:    b.verbose,
	}, nil
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
