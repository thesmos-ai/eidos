// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

// Plugin is the marker interface every plugin satisfies. The Name
// method returns a stable identifier the pipeline uses for
// registration, ordering tie-breaks, error attribution in
// diagnostics, and cache-key derivation.
//
// Names must be unique within a single pipeline; the builder rejects
// duplicates with a positioned diagnostic. Plugin authors should
// pick a stable, repo-namespaced name (e.g. "shape-writer",
// "repogen", "validation") that the user will see in logs and
// configuration.
type Plugin interface {
	// Name returns the plugin's stable identifier.
	Name() string
}
