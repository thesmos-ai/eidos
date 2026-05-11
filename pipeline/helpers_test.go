// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

// assertNoError fails the test if err is non-nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
