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
// # Language parameter
//
// [FilenameSuffix] takes the active backend's language so one
// plugin can ship language-paired surfaces — templates plus
// filename suffix — without committing to a single backend at
// compile time. The shape mirrors [TemplateProvider.Templates],
// which already takes a language argument and returns
// `(fs.FS, bool)` so a plugin's template set narrows by
// language. The matching convention here: return a non-empty
// suffix for every language the plugin supports; return the
// empty string for languages the plugin doesn't ship templates
// for. The framework treats an empty return for the active
// language the same as a plugin not implementing
// FilenameProvider at all.
//
// # When to implement
//
// A generator that emits routable decls or File-level slot
// contributions MUST implement FilenameProvider so the Layout
// phase has a suffix to compose against. The Layout phase emits
// the typed routing error
// [go.thesmos.sh/eidos/pipeline.ErrMissingFilenameProvider] when
// it tries to compose a Filename for a decl attributed to a plugin
// that does not implement the capability — or whose
// FilenameSuffix returns the empty string for the run's
// language.
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
// Returning the empty string for the active language is not a
// valid declaration for a plugin that emits routable output:
// the resulting composed filename would be the bare source
// basename with no extension and would overwrite the source
// file on disk. Generators that have legitimately no filename
// to declare must not implement FilenameProvider at all — that's
// the "method-slot weaver" signal documented above. Plugins
// supporting multiple output families (e.g. a mock generator
// with a test-mode variant) return the per-current-mode suffix
// for the active language and switch the returned value via
// their options.
type FilenameProvider interface {
	Plugin

	// FilenameSuffix returns the trailing identifier the pipeline
	// appends to a source file's basename to form the rendered
	// output filename in the named language — e.g. `_repo.go`
	// for a repository generator targeting `golang`, `_repo.rs`
	// for the same generator targeting `rust`. The leading
	// underscore is conventional but not required; the pipeline
	// appends the returned string verbatim. Returning the empty
	// string for a language signals "this plugin emits no
	// routable output for the named language" — see the
	// "Language parameter" and "Empty-suffix semantics"
	// docblock sections above.
	FilenameSuffix(lang string) string
}
