// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"embed"
	"errors"
	"text/template"

	"go.thesmos.sh/eidos/emit"
)

// templatesFS embeds the backend's core template files. Plugin-
// supplied templates are merged into the same set at render time
// via [plugin.TemplateProvider.Templates].
//
//go:embed templates/*.tmpl
var templatesFS embed.FS

// loadTemplates parses every core template file from [templatesFS]
// into a fresh template set and returns the root. Parse-time
// signature placeholders are registered for every funcmap entry
// the templates reference; each per-Target [renderState] clones
// this root and overrides the placeholders with closures binding
// the correct [writer.ImportSet] for that Target.
//
// Panics on parse failure: the templates are embedded at compile
// time, so a parse error indicates a developer-side syntax bug in
// the templates rather than a runtime condition.
func loadTemplates() *template.Template {
	return template.Must(
		template.New("eidos.golang").
			Funcs(parsePlaceholders).
			ParseFS(templatesFS, "templates/*.tmpl"),
	)
}

// parsePlaceholders satisfies the parser's name-and-arity check
// for funcmap entries referenced by core templates. The bodies
// return [errPlaceholderInvoked] — never executed in practice
// because every Target render clones the root and overrides them
// via [renderState.funcMap] before any ExecuteTemplate call. The
// placeholders match the signatures of the real implementations.
//
//nolint:gochecknoglobals // parse-time signature table; immutable.
var parsePlaceholders = map[string]any{
	"render":           func(emit.Node) (string, error) { return "", errPlaceholderInvoked },
	"renderType":       func(emit.Ref) (string, error) { return "", errPlaceholderInvoked },
	"renderDocs":       func([]string) string { return "" },
	"renderFields":     func([]*emit.Field) (string, error) { return "", errPlaceholderInvoked },
	"renderEmbeds":     func([]*emit.Embed) (string, error) { return "", errPlaceholderInvoked },
	"renderTypeParams": func([]*emit.TypeParam) (string, error) { return "", errPlaceholderInvoked },
	"renderParams":     func([]*emit.Param) (string, error) { return "", errPlaceholderInvoked },
	"renderReceiver":   func(*emit.Method) (string, error) { return "", errPlaceholderInvoked },
	"renderReturns":    func([]emit.Ref) (string, error) { return "", errPlaceholderInvoked },
	"renderExpr":       func(*emit.Expr) (string, error) { return "", errPlaceholderInvoked },
	"renderStmt":       func(*emit.Stmt) (string, error) { return "", errPlaceholderInvoked },
	"renderStmts":      func([]*emit.Stmt) (string, error) { return "", errPlaceholderInvoked },
	"renderVariants":   func(*emit.Enum) (string, error) { return "", errPlaceholderInvoked },
	"imp":              func(string) (string, error) { return "", errPlaceholderInvoked },
}

// errPlaceholderInvoked surfaces if a placeholder closure is ever
// reached at execute time. Indicates a backend bug — execution
// without a per-Target clone of the template set.
var errPlaceholderInvoked = errors.New("backend/golang: template func placeholder invoked; clone before execute")
