// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

import "go.thesmos.sh/eidos/core/directive"

// DirectiveProvider is the optional plugin role for declaring the
// `+gen:` directive schemas the plugin understands. The pipeline
// collects every DirectiveProvider's schemas at Build time and
// registers them centrally with the shared
// [directive.Registry]; collisions across plugins surface as
// [pipeline.ErrDuplicateDirective] (or its underlying registry
// sentinel) so two plugins cannot silently claim the same
// directive name.
//
// Plugins that don't need bespoke directives leave the interface
// unimplemented — the pipeline skips collection for them.
type DirectiveProvider interface {
	Plugin

	// Directives returns every directive schema the plugin
	// declares. The slice is consumed once at Build; mutating it
	// afterward has no effect on the registry.
	Directives() []directive.Schema
}
