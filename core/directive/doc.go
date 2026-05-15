// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package directive parses, validates, and registers `+gen:` /
// `-gen:` source-comment directives.
//
// A [Directive] is one parsed directive instance attached to a source
// node. A [Schema], constructed via [NewSchema], declares the contract
// for a single directive name — which node kinds it may attach to,
// which other directives it requires or excludes, what positional and
// key/value arguments it accepts, and whether it permits the negated
// `-gen:` form.
//
// A [Registry] holds schemas keyed by name. Two plugins registering
// the same name produce [ErrSchemaConflict] at registration time, not
// silent shadowing. Unknown names resolve via [Registry.Suggest] for
// "did you mean?" diagnostics.
//
// Parsing accepts both `+gen:NAME args…` and `-gen:NAME args…` forms.
// Arguments are whitespace-separated; values may be double-quoted to
// embed spaces or `=` literals, with `\"`, `\\`, `\n`, and `\t`
// escapes. Tokens of the form `KEY=VALUE` map to the directive's
// key/value pairs; bare tokens are positional arguments.
//
// Validation against a schema emits positioned [diag] entries
// rather than threaded errors — every failure surfaces with file:line
// context and plugin attribution. Cross-cutting checks
// (Requires / MutuallyExclusiveWith) run over the full set of
// directives on a node via [Validate], not one-directive-at-a-time.
//
// The package owns the [Name] string-aliased type and imports
// the neutral [go.thesmos.sh/eidos/core/kind] package for the
// [kind.Kind] type [Schema.AppliesTo] scopes against; the type
// itself lives outside this package so the source-side [node]
// and emit-side [emit] packages can declare their own kind
// constants without circular imports.
package directive
