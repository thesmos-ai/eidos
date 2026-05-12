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
// Currently:
//
//   - `out`: one positional arg (the filename). Applies to any node
//     kind. The Layout phase reads it off the originating node and
//     stamps [emit.Target.Filename]. Source-side use:
//
//     //+gen:out user_repo_mock.go
//     type UserRepo interface { ... }
func coreDirectives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(OutDirective).
			Describe("Sets the rendered filename for any emit entity originating from the annotated source node. Overrides plugin defaults; CLI -o flag overrides this in turn.").
			Positional("filename").
			DenyNegation().
			Build(),
	}
}
