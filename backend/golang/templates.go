// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"text/template"

	"go.thesmos.sh/eidos/emit"
)

// templatesFS embeds the backend's core template files. Plugin-
// supplied templates are merged into the same set at render time
// via [plugin.TemplateProvider.Templates].
//
//go:embed templates/*.tmpl
var templatesFS embed.FS

// ErrTemplateMissing is returned by render-dispatch helpers when no
// template is registered for an entity's [emit.Node.Kind]. The
// wrapping error names the offending kind so the diagnostic is
// actionable without a stack trace.
var ErrTemplateMissing = errors.New("backend/golang: no template registered for kind")

// loadTemplates parses every core template file from [templatesFS]
// into a fresh template set, attaches the merged funcmap, and
// returns the root [*template.Template]. Child templates resolve
// through the same set, so `{{ render . }}` dispatches across
// arbitrary kinds within the set.
//
// The render-dispatch closure binds the resulting root through a
// shared state struct so the closure and the template set are
// mutually visible without a chicken-and-egg parse-vs-execute
// dependency.
//
// Panics on parse failure: the templates are embedded at compile
// time, so a parse error indicates a developer-side syntax bug in
// the templates rather than a runtime condition.
func loadTemplates() *template.Template {
	state := &templateState{}
	root := template.Must(
		template.New("eidos.golang").
			Funcs(state.funcMap()).
			ParseFS(templatesFS, "templates/*.tmpl"),
	)
	state.root = root
	return root
}

// templateState carries the post-parse [*template.Template] root so
// the funcmap's render-dispatch closure can recurse into it. The
// indirection breaks the chicken-and-egg between funcmap
// registration (must precede parse) and template-set assembly
// (yields the root the closures need).
type templateState struct {
	root *template.Template
}

// funcMap returns the canonical core funcmap with the render-
// dispatch closure bound to the state struct. Calling before
// [templateState.root] is populated still returns a working
// funcmap; the closure surfaces [ErrTemplateMissing] for any
// dispatch attempted before parsing completes.
func (s *templateState) funcMap() template.FuncMap {
	return template.FuncMap{
		"render":     s.render,
		"renderType": renderType,
	}
}

// render executes the template named after n's [emit.Node.Kind] and
// returns the rendered text inline. Returns [ErrTemplateMissing]
// wrapped with the kind when no template is registered.
func (s *templateState) render(n emit.Node) (string, error) {
	kind := string(n.Kind())
	if s.root.Lookup(kind) == nil {
		return "", fmt.Errorf("%w: %s", ErrTemplateMissing, kind)
	}
	var buf bytes.Buffer
	if err := s.root.ExecuteTemplate(&buf, kind, n); err != nil {
		return "", fmt.Errorf("backend/golang: render %s: %w", kind, err)
	}
	return buf.String(), nil
}
