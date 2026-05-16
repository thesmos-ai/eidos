// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mocktest

import (
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/plugins/generator/mock"
	"go.thesmos.sh/eidos/sdk"
)

// Name is the plugin's stable identifier — surfaced in diagnostics,
// cache-key composition, and capability ordering.
const Name = "mocktest"

// Language is the target-language identifier the plugin contributes
// to. Test emission is Go-only today.
const Language = "golang"

// FilenameSuffix is the default per-source-interface filename suffix
// the routing layer appends to the source file's basename. Yields
// `<src-basename>_mock_test.go` next to the source by default;
// configurable via [Options.Suffix].
const FilenameSuffix = "_mock_test.go"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// Suffix overrides the per-source filename suffix used by the
	// Go backend. Empty falls back to [FilenameSuffix].
	Suffix string `eidos:"test_filename_suffix,default=_mock_test.go"`
}

// Plugin is the production mocktest generator. Construct via [New];
// the zero value is unusable because the options holder is nil.
type Plugin struct {
	*opt.Holder[Options]
	opts Options
}

// New returns a ready-to-register plugin with default options.
func New() *Plugin {
	p := &Plugin{}
	p.Holder = opt.Bind(&p.opts)
	return p
}

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Priority places the plugin in the cross-cutting bucket — it
// observes the mock generator's output and emits alongside.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorCrossCutting }

// Provides advertises the `mocktest` capability so dependent
// plugins (recorded-replay generators, property-test extenders) can
// declare a `Requires` edge.
func (*Plugin) Provides() []string { return []string{Name} }

// Requires returns the mock capability — the plugin emits against
// the mock plugin's output and must run after it.
func (*Plugin) Requires() []string { return []string{mock.Capability} }

// FilenameSuffix returns the per-source-interface suffix the
// routing layer appends. Returns the empty string for any language
// other than Go.
func (p *Plugin) FilenameSuffix(lang string) string {
	if lang != Language {
		return ""
	}
	return p.opts.Suffix
}

// Generate walks every emit struct stamped with [mock.MetaIface]
// and emits one test function per dispatch method. The per-mock
// work is in [Plugin.emitTests]; this method only owns the
// gate, per-target package wiring, and store update.
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
	for _, s := range ctx.Reader.EmitStructs().Slice() {
		if _, ok := mock.MetaIface.Get(s.Meta()); !ok {
			continue
		}
		pkg := builder.For(Name, emit.Target{}).
			Package(s.Package, s.Package)
		p.emitTests(pkg, s)
		out, err := pkg.Build()
		if err != nil {
			return err
		}
		if err := ctx.Store.Emit().AddPackage(out); err != nil {
			return err
		}
	}
	return nil
}
