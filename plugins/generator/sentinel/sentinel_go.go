// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sentinel

import (
	"embed"
	"io/fs"
	"text/template"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
)

// GoSuffix is the per-source-basename suffix the Go adapter
// reports through [Plugin.Outputs]. The `_test.go` ending
// triggers the Go backend's automatic `<pkg>_test` package
// shift so the rendered tests live in an external test
// package and can't accidentally read private state.
const GoSuffix = "_sentinel_test.go"

//go:embed templates/golang/*.tmpl
var goTemplatesFS embed.FS

// GoOutputs returns the Go adapter's output set — a single
// `<basename>_sentinel_test.go` file per annotated source
// package the Layout phase routes contributions to.
func GoOutputs() []sdk.Output {
	return []sdk.Output{{Suffix: GoSuffix}}
}

// GoTemplates returns the Go adapter's embedded template
// tree. The Go backend reads it once at Build time and
// registers every `*.tmpl` it finds under
// `templates/golang/`.
func GoTemplates() (fs.FS, bool) {
	sub, _ := fs.Sub(goTemplatesFS, "templates/"+golang.Language)
	return sub, true
}

// GoFuncMap returns nil — the sentinel plugin's templates
// rely only on the canonical `renderType` / `renderExpr` /
// `renderTypeParams` entries (always available) plus the
// shared Go-convention helpers, both of which ride on the
// Go backend's funcmap surface. The plugin contributes no
// extension entries of its own today.
func GoFuncMap() template.FuncMap {
	return nil
}
