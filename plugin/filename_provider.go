// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

// FilenameProvider is the optional capability generators implement
// to declare the filename suffix the pipeline applies to every
// decl they emit. The Layout phase composes
// `<src-basename><Suffix>` when neither the CLI nor a `+gen:out`
// directive pins an explicit filename for the originating decl.
//
// The capability deliberately keeps plugins out of layout
// decisions: a generator never constructs an [emit.Target] itself,
// it only declares what its output is *called*. The pipeline owns
// where output lives (directory, package, filename overrides) so
// the same plugin works across alongside-source and centralised
// layouts and across CLI `-o` / `-p` overrides without per-plugin
// branches.
//
// # When to implement
//
// A generator that emits routable decls or File-level slot
// contributions MUST implement FilenameProvider so the Layout
// phase has a suffix to compose against. The Layout phase emits
// the typed routing error
// [go.thesmos.sh/eidos/pipeline.ErrMissingFilenameProvider] when
// it tries to compose a Filename for a decl attributed to a plugin
// that does not implement the capability.
//
// Pure method-slot weavers — plugins that only contribute items
// to Prebody / Postbody slots on decls some *other* plugin owns
// — must NOT implement FilenameProvider. Their absence of the
// capability is the signal "I never own a routable decl"; growing
// such a plugin to emit a routable decl requires adding
// FilenameProvider at the same change so the Layout phase has a
// suffix to compose against.
//
// # Empty-suffix semantics
//
// Returning the empty string is not a valid declaration: the
// resulting composed filename would be the bare source basename
// with no extension and would overwrite the source file on disk.
// Generators that have legitimately no filename to declare must
// not implement FilenameProvider at all — that's the
// "method-slot weaver" signal documented above. Plugins
// supporting multiple output families (e.g. a mock generator
// with a test-mode variant) return the per-current-mode suffix
// here and switch the returned value via their options.
type FilenameProvider interface {
	Plugin

	// FilenameSuffix returns the trailing identifier the pipeline
	// appends to a source file's basename to form the rendered
	// output filename — e.g. `_repo.go` for a repository generator
	// or `_mock.go` for a mock generator. The leading underscore is
	// conventional but not required; the pipeline appends the
	// returned string verbatim. The empty string is not a valid
	// return value — see the "Empty-suffix semantics" docblock
	// section above.
	FilenameSuffix() string
}
