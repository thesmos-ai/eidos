// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import "go.thesmos.sh/eidos/core/directive"

// OutDirective is the name of the canonical `out` directive — the
// per-source filename override the Layout phase consumes. Source
// authors annotate an entity with `+gen:out filename.go` (or
// whatever prefix the directive parser is configured with) to pin
// the rendered filename for whatever the plugins emit from that
// entity. The directive name is brand-independent; the parser
// strips the prefix before dispatch.
const OutDirective directive.Name = "out"

// coreDirectives returns the directive schemas the pipeline registers
// unconditionally, before any builder-supplied or plugin-supplied
// schemas. These are framework-level directives the Layout phase
// and other internal subsystems depend on; reserving them centrally
// prevents accidental collisions with plugin-defined directives.
//
// The `out` directive supports three forms:
//
//   - `//+gen:out filename.go` — pins the rendered filename for
//     every plugin's output anchored to this source node.
//
//   - `//+gen:out filename.go plugin=<name>` — pins the filename
//     only for the named plugin's output; other plugins fall back
//     to their default suffix or to an unscoped `+gen:out` on the
//     same source. Useful when one plugin's output (mockgen's
//     test-package mode) needs a different filename from the rest.
//
//   - `//+gen:out filename.go pkg=<name>` — pins the rendered
//     `package <name>` clause along with the filename. Combine
//     with `plugin=<name>` to scope the package override too.
//
// `plugin` and `pkg` are key=value modifiers; the filename is the
// positional first argument. Multiple `+gen:out` directives may
// appear on one node — each plugin-scoped form wins over the
// unscoped form for its named plugin, and the unscoped form
// applies to every plugin no scoped directive targets.
//
// Source-side use:
//
//	//+gen:out user_codegen.go
//	//+gen:out user_mock_test.go plugin=mockgen
//	type UserRepo interface { ... }
//
// In that example repogen / buildergen / registrygen land in
// user_codegen.go (`package <src>`); mockgen lands in
// user_mock_test.go (`package <src>_test`). The two routed targets
// have distinct filenames so the one-file-one-package invariant
// stays clean.
func coreDirectives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(OutDirective).
			Describe(
				"Pins the rendered filename for emit entities anchored to this source node. " +
					"Positional `filename` is required; optional `plugin=<name>` scopes the override " +
					"to one plugin's output, and optional `pkg=<name>` pins the rendered package " +
					"clause. CLI -o overrides this directive in turn.",
			).
			Positional("filename").
			DenyNegation().
			Build(),
	}
}
