// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package priority

// Priority is the bucket value plugins return from
// [plugin.CapabilityProvider.Priority]. The pipeline orders buckets
// strictly by ascending Priority; within a bucket, plugins are
// resolved by capability topo sort over Provides / Requires (with
// alphabetical tie-break for determinism).
//
// Lower values run earlier. Plugins can pick predefined values
// (recommended for the standard phases) or custom integers when
// they need a bucket that lives between or outside the conventions.
type Priority int

// Annotator-phase predefined buckets. Annotators land in one of
// these bucket values by convention; plugins may pick custom values
// when they need to order before / after the standard buckets.
const (
	// AnnotatorShape is the bucket for shape-detector annotators —
	// Writer, Reader, Pure, Paginator, Iterator and similar pattern
	// recognisers that stamp shape.* metadata.
	AnnotatorShape Priority = 200

	// AnnotatorRefinement is the bucket for annotators that refine
	// shape facts with additional context (e.g. inferring iterator
	// element types from method signatures).
	AnnotatorRefinement Priority = 300

	// AnnotatorValidation is the bucket for annotators that emit
	// invariant diagnostics — they read prior shape and refinement
	// facts to detect contradictions.
	AnnotatorValidation Priority = 400
)

// Generator-phase predefined buckets. Generators land in one of
// these bucket values by convention.
const (
	// GeneratorFoundation is the bucket for generators that produce
	// base types and interfaces other generators may extend
	// (Repository, Model, …).
	GeneratorFoundation Priority = 100

	// GeneratorComposition is the bucket for generators that depend
	// on foundation output (Builder, Converter, Mock around an
	// existing interface, …).
	GeneratorComposition Priority = 200

	// GeneratorCrossCutting is the bucket for generators that
	// inject into existing emit through slot APIs (Validation,
	// Audit, Debug, Log, Metrics).
	GeneratorCrossCutting Priority = 300

	// GeneratorFinalize is the bucket for last-pass tweaks
	// (formatting hints, comment polishing).
	GeneratorFinalize Priority = 400
)

// Default is the bucket value the pipeline assigns to plugins that
// do not implement [plugin.CapabilityProvider]. It places the plugin
// in the cross-cutting / generator-default bucket and has it run in
// registration order within that bucket.
const Default Priority = 300
