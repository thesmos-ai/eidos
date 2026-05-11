// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

// Phase identifies one execution phase the pipeline runs. Used by
// [Builder.WithParallel] to opt phases into within-bucket parallel
// execution. Sequential execution remains the default so silent
// runs and early-development logs stay simple.
type Phase int

// Phase values in execution order.
const (
	// PhaseFrontend is the source-loading phase. Frontends run
	// independently against patterns; opting in parallelises the
	// frontend×pattern combinations.
	PhaseFrontend Phase = iota
	// PhaseAnnotator is the metadata-stamping phase. Opting in
	// parallelises within-bucket annotators whose declared
	// [plugin.CapabilityProvider.Provides] sets are pairwise
	// disjoint — plugins with overlapping writes still serialise.
	PhaseAnnotator
	// PhaseGenerator is the emit-production phase. Opting in
	// parallelises within-bucket generators that implement
	// [plugin.NodesOnly] (i.e. those that promise to read only the
	// source side of the store, never the emit graph).
	PhaseGenerator
)

// String returns the lower-case textual form of p for diagnostics.
func (p Phase) String() string {
	switch p {
	case PhaseFrontend:
		return "frontend"
	case PhaseAnnotator:
		return "annotator"
	case PhaseGenerator:
		return "generator"
	default:
		return "phase(?)"
	}
}
