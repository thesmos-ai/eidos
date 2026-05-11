// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin

import (
	"io/fs"
	"text/template"
)

// TemplateProvider is the optional capability for plugins that ship
// templates and / or extend a backend's template func-map. The
// backend collects every TemplateProvider it sees, parses templates
// from each provider's filesystem into the rendering tree, and
// merges func-maps before rendering begins.
//
// All three methods are language-scoped via the lang argument.
// Plugins return ("", false) for [TemplateProvider.Templates] when
// they have no templates for the requested language; func-map
// methods return nil to indicate "nothing to add".
type TemplateProvider interface {
	// Templates returns a filesystem of template files for the
	// requested language and a boolean indicating whether the
	// plugin contributes templates to that language. The backend
	// parses every "*.tmpl" file in the returned filesystem.
	Templates(lang string) (fs.FS, bool)

	// TemplateFuncs returns func-map extensions to register for the
	// requested language. The backend rejects extensions whose
	// names collide with previously-registered names from another
	// plugin (use [TemplateProvider.TemplateOverrides] for
	// intentional override).
	TemplateFuncs(lang string) template.FuncMap

	// TemplateOverrides returns func-map entries that intentionally
	// replace previously-registered names. The backend records each
	// override as a diagnostic so users can see which plugin's
	// definition won.
	TemplateOverrides(lang string) template.FuncMap
}
