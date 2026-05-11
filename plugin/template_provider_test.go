// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package plugin_test

import (
	"testing"
	"testing/fstest"
	"text/template"

	"go.thesmos.sh/eidos/plugin"
)

func TestTemplateProvider_Contract(t *testing.T) {
	t.Parallel()

	t.Run("Templates returns the supplied filesystem and a true flag when set", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{"a.tmpl": &fstest.MapFile{Data: []byte("hi")}}
		var tp plugin.TemplateProvider = &stubTemplate{name: "stub", tmplFS: fsys}
		got, ok := tp.Templates("go")
		if !ok {
			t.Fatalf("Templates should return true when an fs is supplied")
		}
		if _, err := got.Open("a.tmpl"); err != nil {
			t.Fatalf("returned fs should expose the supplied entries; got %v", err)
		}
	})

	t.Run("Templates returns false when the plugin contributes nothing for the language", func(t *testing.T) {
		t.Parallel()
		var tp plugin.TemplateProvider = &stubTemplate{name: "stub"}
		_, ok := tp.Templates("go")
		if ok {
			t.Fatalf("Templates should return false when no fs is supplied")
		}
	})

	t.Run("TemplateFuncs and TemplateOverrides return the configured func-maps", func(t *testing.T) {
		t.Parallel()
		funcs := template.FuncMap{"upper": func() string { return "" }}
		overs := template.FuncMap{"lower": func() string { return "" }}
		var tp plugin.TemplateProvider = &stubTemplate{name: "stub", funcs: funcs, overrides: overs}
		if _, ok := tp.TemplateFuncs("go")["upper"]; !ok {
			t.Fatalf("TemplateFuncs should expose registered names")
		}
		if _, ok := tp.TemplateOverrides("go")["lower"]; !ok {
			t.Fatalf("TemplateOverrides should expose registered names")
		}
	})
}
