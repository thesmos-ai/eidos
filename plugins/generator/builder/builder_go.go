// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
)

// GoSuffix is the per-source-basename suffix the Go adapter
// reports through [Plugin.Outputs]. All `+gen:builder`
// structs declared in `article.go` collate into a single
// `article_builder.go`.
const GoSuffix = "_builder.go"

// ErrMalformedDefaults is the sentinel surfaced by
// [GoDefaultsExpr] when a `defaults=` value cannot be split
// into a non-empty import path + non-empty function
// identifier (Go's `<import-path>.<FuncName>` convention).
// Template execution fails with this error wrapped so the
// offending struct's whole render attempt errors out rather
// than silently dropping the `New<Name>WithDefaults` branch.
var ErrMalformedDefaults = errors.New(
	`builder: defaults value must have form "import/path.FuncName"`,
)

//go:embed templates/golang/*.tmpl
var goTemplatesFS embed.FS

// GoOutputs returns the Go adapter's output set — a single
// `<basename>_builder.go` file per source-file the Layout
// phase routes contributions to.
func GoOutputs() []sdk.Output {
	return []sdk.Output{{Suffix: GoSuffix}}
}

// GoTemplates returns the Go adapter's embedded template
// tree. The Go backend reads it once at Build time and
// registers every `*.tmpl` it finds.
func GoTemplates() (fs.FS, bool) {
	sub, _ := fs.Sub(goTemplatesFS, "templates/"+golang.Language)
	return sub, true
}

// GoFuncMap returns the builder-specific Go funcmap entries
// the `templates/golang/builder.type.tmpl` template consumes
// — only `defaultsExpr` today. Shared Go-convention helpers
// (`isExported`, `isByteSlice`, `selfType`, ...) ride on the
// Go backend's canonical extras funcmap and are available to
// every Go template without per-plugin registration.
func GoFuncMap() template.FuncMap {
	return template.FuncMap{
		"defaultsExpr": GoDefaultsExpr,
	}
}

// GoDefaultsExpr parses a Go-shape `<import-path>.<FuncName>`
// defaults value into an [emit.External] expression suitable
// for the rendered `New<Name>WithDefaults` body. Malformed
// input — empty string, no `.`, leading `.`, trailing `.` —
// returns [ErrMalformedDefaults] wrapped with the offending
// value, surfaced as a render-time error.
//
// The template guards the call with `{{if .DefaultsArg}}` so
// the empty case never reaches the parser under normal
// rendering; the empty-input rejection is defence-in-depth
// for direct callers.
//
// The parser is plugin-local because the directive's
// "import-path.FuncName" convention is specific to the
// builder's `defaults=` arg; shared identifier-convention
// helpers live in [golang].
func GoDefaultsExpr(raw string) (*sdk.Expr, error) {
	i := strings.LastIndex(raw, ".")
	if i <= 0 || i == len(raw)-1 {
		return nil, fmt.Errorf("%w (got %q)", ErrMalformedDefaults, raw)
	}
	return sdk.NewExternal(raw[:i], raw[i+1:]), nil
}
