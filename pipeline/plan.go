// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import "go.thesmos.sh/eidos/plugin"

// Plan is the resolved execution order produced by [Builder.Build].
// Frontends run in registration order (no priority on the frontend
// role); annotators and generators are grouped into priority
// buckets and topo-sorted within each bucket using
// [plugin.CapabilityProvider.Provides] / [plugin.CapabilityProvider.Requires]
// with alphabetical tie-break for determinism. The Backend slot
// holds the single registered backend.
//
// The slice fields are already in execution order — callers iterate
// front-to-back. The plan is also exposed via [Pipeline.Plan] for
// "eidos explain plan" tooling that wants to display the resolved
// ordering without running the pipeline.
type Plan struct {
	Frontends  []plugin.Frontend
	Annotators []plugin.Annotator
	Generators []plugin.Generator
	Backend    plugin.Backend
}
