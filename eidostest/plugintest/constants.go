// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugintest

// Package-private sentinel strings used in failure-output
// projections. Centralised so the goconst linter does not flag
// repeated literal occurrences across the role-specific suite
// files.
const (
	// unnamedSentinel is the placeholder used when a node's
	// owner / identity reflectively exposes neither QName nor
	// Name. Surfaces in projection output so unnamed entries
	// remain greppable.
	unnamedSentinel = "<unnamed>"

	// unownedSentinel is the placeholder used when a node's
	// owner back-pointer is nil. Surfaces in projection output
	// so unowned entries remain greppable.
	unownedSentinel = "<unowned>"

	// emptyTargetSentinel is the placeholder used for empty
	// [emit.Target] fields (Dir / Filename / Package) so a
	// missing routing slot stays visible in failure output.
	emptyTargetSentinel = "<empty>"
)
