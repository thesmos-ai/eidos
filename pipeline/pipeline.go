// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"sync/atomic"

	"go.thesmos.sh/eidos/cache"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// Pipeline is the validated, ready-to-run artifact returned by
// [Builder.Build]. It holds the participating plugins grouped by
// role plus the shared sink, cache, and diagnostic sink supplied
// during construction, and the resolved execution-ordered [Plan].
//
// A Pipeline is immutable after Build; access to its members is
// read-only and callers must not mutate the slices they receive
// from the typed accessors. Future milestones add Run / DryRun
// methods that execute the phases in plan order.
type Pipeline struct {
	frontends    []plugin.Frontend
	annotators   []plugin.Annotator
	generators   []plugin.Generator
	backend      plugin.Backend
	sink         sink.Sink
	cache        cache.Cache
	diag         *diag.Sink
	verbose      bool
	parallel     map[Phase]bool
	manifestPath string
	command      string
	sourceRoot   string
	plan         *Plan
	registry     *directive.Registry
	parser       *directive.Parser

	// lastStore caches the store from the most recent Run for
	// post-run inspection (test harnesses, "eidos explain"
	// tooling). Stored under an atomic.Pointer so concurrent
	// readers — typically tests dispatching multiple parallel
	// runs against the same Pipeline — observe a coherent value
	// without locking the run path.
	lastStore atomic.Pointer[store.Store]
}

// Store returns the [store.Store] used by the most recent
// [Pipeline.Run] invocation, or nil when Run has not been called.
// The returned store is read-only from the caller's perspective:
// the pipeline freezes the node + emit views as it transitions
// between phases, and reusing the Pipeline for another Run replaces
// the cached store with the next run's instance.
func (p *Pipeline) Store() *store.Store { return p.lastStore.Load() }

// Frontends returns the registered frontends in registration order.
func (p *Pipeline) Frontends() []plugin.Frontend {
	out := make([]plugin.Frontend, len(p.frontends))
	copy(out, p.frontends)
	return out
}

// Annotators returns the registered annotators in registration
// order. For the resolved execution order (priority bucket +
// capability topo) call [Pipeline.Plan] and read its Annotators
// field.
func (p *Pipeline) Annotators() []plugin.Annotator {
	out := make([]plugin.Annotator, len(p.annotators))
	copy(out, p.annotators)
	return out
}

// Generators returns the registered generators in registration
// order. For the resolved execution order call [Pipeline.Plan].
func (p *Pipeline) Generators() []plugin.Generator {
	out := make([]plugin.Generator, len(p.generators))
	copy(out, p.generators)
	return out
}

// Backend returns the single registered backend.
func (p *Pipeline) Backend() plugin.Backend { return p.backend }

// Sink returns the destination sink the backend writes through.
// Returns nil when no sink was configured (the pipeline run loop
// later in this milestone surfaces a diagnostic in that case).
func (p *Pipeline) Sink() sink.Sink { return p.sink }

// Cache returns the configured [cache.Cache]. Returns the
// no-op [cache.None] sentinel when no cache was configured.
func (p *Pipeline) Cache() cache.Cache { return p.cache }

// Diag returns the diagnostic sink shared across every phase.
func (p *Pipeline) Diag() *diag.Sink { return p.diag }

// Verbose reports whether verbose-mode diagnostics are enabled.
func (p *Pipeline) Verbose() bool { return p.verbose }

// Plan returns the resolved execution order — annotators and
// generators grouped into priority buckets and topo-sorted within
// each bucket. "eidos explain plan" tooling reads this to display
// the resolved ordering without running the pipeline.
func (p *Pipeline) Plan() *Plan { return p.plan }

// DirectiveRegistry returns the [directive.Registry] populated at
// Build time from every schema supplied via [Builder.WithDirective].
// Frontends consult the registry while parsing source comments to
// validate every directive against its schema; tooling can also
// enumerate registered directives for documentation.
func (p *Pipeline) DirectiveRegistry() *directive.Registry { return p.registry }
