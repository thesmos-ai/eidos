// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

import "go.thesmos.sh/eidos/core/directive"

// DirectiveProvider is the optional capability for plugins that
// auto-register directive schemas at pipeline Build time. The
// pipeline collects [DirectiveProvider.Directives] from every
// participating plugin and registers each schema with the central
// directive registry; collisions surface as positioned diagnostics
// referencing both registering plugins.
//
// Plugins that ship directives implement this interface so users
// don't have to register schemas manually in configuration. The
// schemas declare AppliesTo, Requires, MutuallyExclusiveWith,
// RequiredKeys, AllowedKeys, PositionalArgs, and
// AllowExtraPositional rules used to validate every directive
// instance the frontend parses.
type DirectiveProvider interface {
	// Directives returns the schemas this plugin registers. Each
	// schema is registered once at Build; the pipeline rejects
	// duplicate names across plugins with positioned diagnostics.
	Directives() []directive.Schema
}
