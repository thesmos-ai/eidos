// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sdk

import "go.thesmos.sh/eidos/priority"

// Priority is the bucket value [CapabilityProvider] plugins
// return from [CapabilityProvider.Priority]. The pipeline orders
// buckets strictly by ascending Priority; within a bucket,
// plugins are topo-sorted via Provides / Requires.
type Priority = priority.Priority

// Canonical priority buckets — annotator and generator phases.
// Plugins return one of these from [CapabilityProvider.Priority];
// project-local plugins may return any [Priority] value, but
// using the standard buckets keeps cross-plugin ordering
// predictable.
const (
	// AnnotatorShape is for annotators that infer structural
	// shape (e.g. "this struct is a Repository"). Runs before
	// AnnotatorRefinement so refinement annotators see a
	// populated shape baseline.
	AnnotatorShape = priority.AnnotatorShape

	// AnnotatorRefinement is for annotators that refine or
	// enrich shape inferences from earlier annotators.
	AnnotatorRefinement = priority.AnnotatorRefinement

	// AnnotatorValidation is for annotators that validate the
	// final annotated state. Runs last in the annotator phase.
	AnnotatorValidation = priority.AnnotatorValidation

	// GeneratorFoundation is for generators that emit the
	// baseline output other generators may compose on.
	// Repository / builder / shape-derived generators live here.
	GeneratorFoundation = priority.GeneratorFoundation

	// GeneratorComposition is for generators that compose
	// foundation output (e.g. a mock generator that depends on
	// foundation-emitted interfaces).
	GeneratorComposition = priority.GeneratorComposition

	// GeneratorCrossCutting is for generators that contribute
	// slot content (audit traces, debug statements, registry
	// entries) into other generators' emit decls.
	GeneratorCrossCutting = priority.GeneratorCrossCutting

	// GeneratorFinalize is for generators that finalise the emit
	// graph (e.g. validation passes, last-mile transforms). Runs
	// last in the generator phase.
	GeneratorFinalize = priority.GeneratorFinalize

	// DefaultPriority is the bucket value plugins implicitly
	// occupy when they do not implement [CapabilityProvider].
	// Equivalent to [GeneratorCrossCutting].
	DefaultPriority = priority.Default
)
