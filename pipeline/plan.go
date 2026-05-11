// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
)

// Plan is the resolved execution order produced by [Builder.Build].
// Frontends run in registration order (no priority on the frontend
// role); annotators and generators are grouped into priority
// buckets and topo-sorted within each bucket using
// [plugin.CapabilityProvider.Provides] / [plugin.CapabilityProvider.Requires]
// with alphabetical tie-break for determinism. The Backend slot
// holds the single registered backend.
//
// Annotators and Generators are the flat sequential execution
// orders across every bucket — convenient for callers that don't
// need bucket-aware iteration. AnnotatorBuckets and
// GeneratorBuckets expose the per-bucket grouping the pipeline
// uses for within-bucket parallel execution.
type Plan struct {
	Frontends []plugin.Frontend

	// Annotators is the flat sequential execution order across
	// every bucket. Equal to the concatenation of every
	// AnnotatorBuckets[i].Plugins in bucket order.
	Annotators []plugin.Annotator

	// AnnotatorBuckets groups annotators by their priority bucket
	// (sorted ascending). Within-bucket order is the topo-sorted
	// execution order; iterate bucket.Plugins sequentially or
	// dispatch them concurrently when [PhaseAnnotator] is opted
	// into via [Builder.WithParallel].
	AnnotatorBuckets []AnnotatorBucket

	// Generators is the flat sequential execution order across
	// every bucket.
	Generators []plugin.Generator

	// GeneratorBuckets groups generators by their priority bucket.
	// Same semantics as AnnotatorBuckets.
	GeneratorBuckets []GeneratorBucket

	// Backend is the single registered backend.
	Backend plugin.Backend
}

// AnnotatorBucket is one priority bucket of annotators in
// topo-sorted execution order. The buckets are visited in
// ascending [priority.Priority] order; within a bucket the plugins
// may run sequentially or in parallel depending on pipeline config.
type AnnotatorBucket struct {
	Priority priority.Priority
	Plugins  []plugin.Annotator
}

// GeneratorBucket mirrors [AnnotatorBucket] for the generator phase.
type GeneratorBucket struct {
	Priority priority.Priority
	Plugins  []plugin.Generator
}
