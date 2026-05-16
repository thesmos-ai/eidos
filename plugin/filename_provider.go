// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

// Output is one entry in a [FilenameProvider]'s ordered output
// set — the plugin's declaration of one rendered file. Single-file
// plugins declare a single Output with an empty Tag and behave
// identically to the framework's pre-multi-output era;
// multi-file plugins declare additional outputs each with a
// distinct, non-empty Tag.
//
// Tag is the stable per-plugin identifier the rest of the
// routing surface (`+gen:out tag=<tag>`, CLI `-o <plugin>:<tag>`,
// project config under the plugin's `tags:` block, manifest
// attribution) keys on. The empty Tag is reserved for the
// plugin's primary output — at most one per plugin, declared at
// index 0 when present. Multi-output plugins that require every
// decl to be explicitly tagged declare every Output with a
// non-empty Tag.
//
// Suffix is the per-source-basename trailer the Layout phase
// appends — composed as `<src-basename><Suffix>` when neither a
// `+gen:out` directive nor a CLI override pins an explicit
// filename for the originating decl. The leading underscore is
// conventional but not required; the pipeline appends the value
// verbatim. The empty Suffix is always invalid.
type Output struct {
	// Tag identifies this output within the plugin's namespace.
	// Empty for the plugin's primary output; non-empty values
	// surface in directive scoping, CLI overrides, project
	// config, and manifest attribution as `<plugin>:<tag>`.
	Tag string

	// Suffix is the per-source-basename trailer the Layout phase
	// appends to compose the rendered output filename. Required
	// and non-empty for every Output a plugin declares.
	Suffix string
}

// FilenameProvider is the optional capability generators implement
// to declare the ordered set of outputs they emit for the active
// backend language. The Layout phase composes
// `<src-basename><Output.Suffix>` per output when neither the CLI
// nor a `+gen:out` directive pins an explicit filename for the
// originating decl.
//
// The capability deliberately keeps plugins out of layout
// decisions: a generator never constructs an [emit.Target] itself,
// it only declares what its outputs are *called*. The pipeline
// owns where output lives (directory, package, filename
// overrides) so the same plugin works across alongside-source and
// centralised layouts and across CLI / directive / config
// overrides without per-plugin branches.
//
// # Language parameter
//
// [Outputs] takes the active backend's language so one plugin can
// ship language-paired surfaces — templates plus output set —
// without committing to a single backend at compile time. The
// shape mirrors [TemplateProvider.Templates], which already takes
// a language argument and narrows the plugin's per-language
// surface to the active backend.
//
// # When to implement
//
// A generator that emits routable decls or File-level slot
// contributions MUST implement FilenameProvider so the Layout
// phase has at least one Output suffix to compose against. The
// Layout phase emits the typed routing error
// [go.thesmos.sh/eidos/pipeline.ErrMissingFilenameProvider] when
// it tries to compose a Filename for a decl attributed to a
// plugin that does not implement the capability — or whose
// Outputs returns an empty slice for the run's language.
//
// Pure method-slot weavers — plugins that only contribute items
// to Prebody / Postbody slots on decls some *other* plugin owns
// — must NOT implement FilenameProvider. Their absence of the
// capability is the signal "I never own a routable decl"; growing
// such a plugin to emit a routable decl requires adding
// FilenameProvider at the same change so the Layout phase has an
// output set to compose against.
//
// # Empty / nil-return semantics
//
// Returning nil or an empty slice for the active language is not
// a valid declaration for a plugin that emits routable output:
// the absence of any Output for the active backend signals "this
// plugin has no routable output for the named language" — the
// framework treats that the same as a non-implementer of
// FilenameProvider. Plugins supporting multiple output families
// (e.g. a mock generator with a test-mode variant) declare each
// family as a separate Output with its own Tag and Suffix.
//
// # Output-shape validation
//
// The framework rejects malformed Outputs slices at Build time
// via the pipeline builder. The validation rules are documented
// alongside the multi-output design; in summary: every Suffix
// must be non-empty; Tags within one slice must be unique; at
// most one Output may declare an empty Tag, and when present the
// empty-Tag Output must be at index 0.
type FilenameProvider interface {
	Plugin

	// Outputs returns the ordered set of rendered files this
	// plugin produces in the named language. Each entry's Suffix
	// is appended to a source file's basename to form the
	// rendered output filename; each entry's Tag identifies the
	// output within the plugin's namespace for directive / CLI
	// / config scoping. Returning nil or an empty slice signals
	// "this plugin emits no routable output for the named
	// language" — see the "Empty / nil-return semantics"
	// docblock section above.
	Outputs(lang string) []Output
}
