// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"io/fs"
	"testing"
	"text/template"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
	"go.thesmos.sh/eidos/store"
)

// assertNoError fails the test if err is non-nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// names extracts the plugin names from a plugin slice for compact
// equality assertions in resolver and plan tests.
func names[T interface{ Name() string }](plugins []T) []string {
	out := make([]string, len(plugins))
	for i, p := range plugins {
		out[i] = p.Name()
	}
	return out
}

// stubFE implements [plugin.Frontend].
type stubFE struct{ name string }

func (f *stubFE) Name() string                                    { return f.name }
func (*stubFE) Load(_ string, _ *store.Store, _ *diag.Sink) error { return nil }

// stubAnn implements [plugin.Annotator].
type stubAnn struct{ name string }

func (a *stubAnn) Name() string                            { return a.name }
func (*stubAnn) Annotate(_ *plugin.AnnotatorContext) error { return nil }

// stubGen implements [plugin.Generator].
type stubGen struct{ name string }

func (g *stubGen) Name() string                            { return g.name }
func (*stubGen) Generate(_ *plugin.GeneratorContext) error { return nil }

// stubBE implements [plugin.Backend].
type stubBE struct{ name string }

func (b *stubBE) Name() string                        { return b.name }
func (*stubBE) Language() string                      { return "stub" }
func (*stubBE) Render(_ *plugin.BackendContext) error { return nil }

// stubFEWithOpts is a frontend that also implements
// [plugin.OptionsProvider]. The configured options struct exposes a
// single required string field; tests use it to drive the
// validation paths in Build.
type stubFEWithOpts struct {
	name string
	opts stubFEOptions
}

type stubFEOptions struct {
	Output string `eidos:"output,required"`
}

func (f *stubFEWithOpts) Name() string                                    { return f.name }
func (*stubFEWithOpts) Load(_ string, _ *store.Store, _ *diag.Sink) error { return nil }
func (*stubFEWithOpts) OptionsSchema() opt.Schema                         { return opt.Reflect(stubFEOptions{}) }

func (f *stubFEWithOpts) SetOptions(o opt.Options) error { return o.Decode(&f.opts) }

// stubAnnCap is an annotator that also implements
// [plugin.CapabilityProvider]. Used to drive bucket grouping and
// topo resolution in Build tests.
type stubAnnCap struct {
	name     string
	priority priority.Priority
	provides []string
	requires []string
}

func (a *stubAnnCap) Name() string                            { return a.name }
func (*stubAnnCap) Annotate(_ *plugin.AnnotatorContext) error { return nil }
func (a *stubAnnCap) Priority() priority.Priority             { return a.priority }
func (a *stubAnnCap) Provides() []string                      { return a.provides }
func (a *stubAnnCap) Requires() []string                      { return a.requires }

// stubGenCap is a generator that also implements
// [plugin.CapabilityProvider]. Mirrors stubAnnCap for the
// generator phase.
type stubGenCap struct {
	name     string
	priority priority.Priority
	provides []string
	requires []string
}

func (g *stubGenCap) Name() string                            { return g.name }
func (*stubGenCap) Generate(_ *plugin.GeneratorContext) error { return nil }
func (g *stubGenCap) Priority() priority.Priority             { return g.priority }
func (g *stubGenCap) Provides() []string                      { return g.provides }
func (g *stubGenCap) Requires() []string                      { return g.requires }

// stubBEWithTemplates is a backend that also implements
// [plugin.TemplateProvider]. Used by template-collision tests to
// drive the validateTemplateFuncs path indirectly through Build.
type stubBEWithTemplates struct {
	name      string
	lang      string
	funcs     template.FuncMap
	overrides template.FuncMap
}

func (b *stubBEWithTemplates) Name() string                                { return b.name }
func (b *stubBEWithTemplates) Language() string                            { return b.lang }
func (*stubBEWithTemplates) Render(_ *plugin.BackendContext) error         { return nil }
func (*stubBEWithTemplates) Templates(_ string) (fs.FS, bool)              { return nil, false }
func (b *stubBEWithTemplates) TemplateFuncs(_ string) template.FuncMap     { return b.funcs }
func (b *stubBEWithTemplates) TemplateOverrides(_ string) template.FuncMap { return b.overrides }

// stubGenWithTemplates is a generator that ships templates for the
// pipeline's backend language. Used to drive cross-plugin template
// func collision tests.
type stubGenWithTemplates struct {
	name      string
	funcs     template.FuncMap
	overrides template.FuncMap
}

func (g *stubGenWithTemplates) Name() string                                { return g.name }
func (*stubGenWithTemplates) Generate(_ *plugin.GeneratorContext) error     { return nil }
func (*stubGenWithTemplates) Templates(_ string) (fs.FS, bool)              { return nil, false }
func (g *stubGenWithTemplates) TemplateFuncs(_ string) template.FuncMap     { return g.funcs }
func (g *stubGenWithTemplates) TemplateOverrides(_ string) template.FuncMap { return g.overrides }
