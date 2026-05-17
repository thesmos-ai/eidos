// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"embed"
	"io/fs"
	"text/template"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
)

// GoPrimarySuffix is the per-source-basename suffix the Go
// adapter reports for the primary output. Source enums
// declared in `status.go` produce `status_enum.go`.
const GoPrimarySuffix = "_enum.go"

// GoTestSuffix is the per-source-basename suffix the Go
// adapter reports for the test-tagged output. The
// `_test.go` ending triggers the Go backend's automatic
// `<pkg>_test` package shift so the rendered tests live in
// an external test package.
const GoTestSuffix = "_enum_test.go"

// GoTestOutputTag is the tag the test-tagged output
// advertises. Source-side `+gen:out tag=test …` directives
// and CLI `-o enum:test=…` overrides match against this
// value.
const GoTestOutputTag = "test"

// GoDefaultParsePrefix is the Go-idiomatic prefix appended
// to the enum's type name to form the parse function's
// identifier when [Options.ParsePrefix] is unset.
// `ParseStatus` matches the canonical Go pattern.
const GoDefaultParsePrefix = "Parse"

// GoDefaultSentinelPrefix is the Go-idiomatic prefix
// appended to the enum's type name to form the parse-error
// sentinel's identifier when [Options.SentinelPrefix] is
// unset. `ErrUnknownStatus` reads as a typed sentinel
// callers compare via [errors.Is].
const GoDefaultSentinelPrefix = "ErrUnknown"

//go:embed templates/golang/*.tmpl
var goTemplatesFS embed.FS

// GoOutputs returns the Go adapter's output set — the
// primary `<basename>_enum.go` file plus the
// "test"-tagged `<basename>_enum_test.go` companion.
func GoOutputs() []sdk.Output {
	return []sdk.Output{
		{Suffix: GoPrimarySuffix},
		{Tag: GoTestOutputTag, Suffix: GoTestSuffix},
	}
}

// GoTemplates returns the Go adapter's embedded template
// tree. The Go backend reads it once at Build time and
// registers every `*.tmpl` it finds under
// `templates/golang/`.
func GoTemplates() (fs.FS, bool) {
	sub, _ := fs.Sub(goTemplatesFS, "templates/"+golang.Language)
	return sub, true
}

// GoFuncMap returns nil — the enum plugin's templates rely
// only on the canonical `renderType` / `renderExpr` /
// `renderTypeParams` entries (always available) plus the
// shared Go-convention helpers, both of which ride on the
// Go backend's funcmap surface. The plugin contributes no
// extension entries of its own today.
func GoFuncMap() template.FuncMap {
	return nil
}
