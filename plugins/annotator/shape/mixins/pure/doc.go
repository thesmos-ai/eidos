// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package pure recognises the pure mixin — the assertion that
// the annotated callable is deterministic, side-effect free, and
// referentially transparent (same inputs always produce same
// outputs). Distinct from the structural [shape/detectors/pure]
// detector, which classifies callables by signature; this mixin
// asserts the behavioural property the test suite checks.
//
// The recognised directive is:
//
//	//+gen:mixin pure
package pure
