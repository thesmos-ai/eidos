// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
)

// testMetaKindKey / testMetaExportedKey are package-level metadata
// keys the meta-funcmap test seeds onto a fixture entity.
// Declared at package scope so the global key registry sees each
// key exactly once — meta.NewKey panics on duplicate names, so
// repeated test runs (e.g. `go test -count=2`) need a single
// registration site.
//
//nolint:gochecknoglobals // shared across subtest invocations; registration is one-shot by design.
var (
	testMetaKindKey     = meta.NewKey("test-kind", meta.StringParser)
	testMetaExportedKey = meta.NewKey("test-exported", meta.BoolParser)
)

// TestFuncmapExtras_Naming covers the Naming-category funcmap
// entries (pascal, camel, snake, screaming, exported) — they
// delegate to `core/naming` so the assertion is on the round-trip
// rendered text rather than the conversion semantics themselves.
func TestFuncmapExtras_Naming(t *testing.T) {
	t.Parallel()

	body := renderWithPluginTemplate(t,
		`{{ define "emit.constant" -}}
// pascal:    {{ pascal "user_id" }}
// camel:     {{ camel "user_id" }}
// snake:     {{ snake "UserID" }}
// screaming: {{ screaming "user_id" }}
// exported:  {{ exported "userID" }}
const {{ .Name }} = {{ renderExpr .Value }}
{{- end -}}`)
	for _, want := range []string{
		"// pascal:    UserID",
		"// camel:     userID",
		"// snake:     user_id",
		"// screaming: USER_ID",
		"// exported:  UserID",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in body; got:\n%s", want, body)
		}
	}
}

// TestFuncmapExtras_String covers the String-category funcmap
// entries (join, title, upper, lower, trim, split, default,
// coalesce).
func TestFuncmapExtras_String(t *testing.T) {
	t.Parallel()

	body := renderWithPluginTemplate(t,
		`{{ define "emit.constant" -}}
// join:     {{ join (split "a,b,c" ",") " | " }}
// upper:    {{ upper "hello" }}
// lower:    {{ lower "WORLD" }}
// trim:     {{ trim "  spaced  " }}
// default:  {{ default "" "fallback" }}
// coalesce: {{ coalesce "" "" "third" }}
const {{ .Name }} = {{ renderExpr .Value }}
{{- end -}}`)
	for _, want := range []string{
		"// join:     a | b | c",
		"// upper:    HELLO",
		"// lower:    world",
		"// trim:     spaced",
		"// default:  fallback",
		"// coalesce: third",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in body; got:\n%s", want, body)
		}
	}
}

// TestFuncmapExtras_Meta covers the Meta-read category — meta,
// metaBool, metaStr, hasMeta, metaEq — by populating an entity's
// metadata bag and asserting the rendered template surfaces the
// values correctly.
func TestFuncmapExtras_Meta(t *testing.T) {
	t.Parallel()

	body := renderWithPluginTemplateUsing(t,
		`{{ define "emit.constant" -}}
// hasMeta:  {{ hasMeta . "test-kind" }}
// metaStr:  {{ metaStr . "test-kind" }}
// metaBool: {{ metaBool . "test-exported" }}
// metaEq:   {{ metaEq . "test-kind" "primary" }}
const {{ .Name }} = {{ renderExpr .Value }}
{{- end -}}`,
		func(c *emit.Constant) {
			testMetaKindKey.SetManual(c.Meta(), "primary", "extrasprobe")
			testMetaExportedKey.SetManual(c.Meta(), true, "extrasprobe")
		})
	for _, want := range []string{
		"// hasMeta:  true",
		"// metaStr:  primary",
		"// metaBool: true",
		"// metaEq:   true",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in body; got:\n%s", want, body)
		}
	}
}

// TestFuncmapExtras_Debug covers the Debug category — origin and
// explain — by rendering an entity through a plugin template that
// prints both helpers. The entity is synthetic (no Origin) so the
// helpers surface their no-provenance forms.
func TestFuncmapExtras_Debug(t *testing.T) {
	t.Parallel()

	body := renderWithPluginTemplate(t,
		`{{ define "emit.constant" -}}
// origin:  {{ origin . }}
// explain: {{ explain . }}
const {{ .Name }} = {{ renderExpr .Value }}
{{- end -}}`)
	// gofmt strips trailing whitespace from comments, so the
	// empty-origin form lands as a bare `// origin:` line.
	if !strings.Contains(body, "// origin:\n") {
		t.Fatalf("expected origin line; got:\n%s", body)
	}
	if !strings.Contains(body, "// explain: emit.constant") {
		t.Fatalf("expected explain line carrying the kind; got:\n%s", body)
	}
}

// renderWithPluginTemplate runs the backend against a single-
// constant fixture rendered through a plugin-supplied
// `emit.constant` template body. Centralised so each per-category
// test stays at the assertion altitude.
func renderWithPluginTemplate(t *testing.T, body string) string {
	t.Helper()
	return renderWithPluginTemplateUsing(t, body, nil)
}

// renderWithPluginTemplateUsing extends renderWithPluginTemplate
// with a configure hook the caller uses to seed metadata or
// other per-entity state before render.
func renderWithPluginTemplateUsing(t *testing.T, body string, configure func(*emit.Constant)) string {
	t.Helper()
	ctx, mem, d := newBackendContext(t)
	provider := &stubTemplateProvider{
		name: "extrasprobe",
		tmplFS: fstest.MapFS{
			"templates/golang/override.tmpl": &fstest.MapFile{Data: []byte(body)},
		},
	}
	ctx.Plugins = []plugin.Plugin{provider}
	ctx.Ordered = []plugin.Plugin{provider}
	target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
	k := &emit.Constant{
		Name: "K", Package: "x", Target: target,
		Value: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitInt, RawText: "1"},
	}
	if configure != nil {
		configure(k)
	}
	addEmitPackage(t, ctx, &emit.Package{
		Name: "x", Path: "x",
		Constants: []*emit.Constant{k},
	})
	return string(assertRenderSucceeds(t, ctx, mem, d, target))
}
