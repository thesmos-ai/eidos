// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mock

import (
	"errors"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"
)

// MetaIface is the meta key the plugin stamps on every emitted
// mock struct, carrying the qualified name of the source
// interface the struct mocks. Companion plugins (e.g.
// `plugins/generator/mocktest`) read it to discover which emit
// structs are mocks worth processing — looser coupling than
// hard-coding the producing plugin's name.
//
//nolint:gochecknoglobals // registry-singleton key
var MetaIface = meta.NewKey("mock.iface", meta.StringParser)

// MetaField is the meta key the plugin stamps on every emitted
// dispatch method, carrying the name of the func-valued struct
// field that backs the method (e.g. method `Get` → field `OnGet`).
// Companion plugins use it to drive the override branch without
// re-deriving the field name from the producing plugin's options.
//
//nolint:gochecknoglobals // registry-singleton key
var MetaField = meta.NewKey("mock.field", meta.StringParser)

// Name is the plugin's stable identifier — used in diagnostics,
// cache-key composition, and capability ordering.
const Name = "mock"

// Capability is the capability label this plugin advertises so
// dependent plugins (recording, fault-injection, …) can declare
// `Requires(mock.Capability)` to order themselves after the
// mock is emitted.
const Capability = "mock"

// DirectiveName is the bare `+gen:` directive name read from
// source interfaces. Positive form opts an interface into mock
// generation; negative form suppresses it.
const DirectiveName sdk.DirectiveName = "mock"

// Language is the target-language identifier whose template
// surface the plugin contributes to. Mocks are Go-only today;
// other-language equivalents register their own variants.
const Language = "golang"

// FilenameSuffix is the per-source-interface filename suffix the
// routing layer appends to the source file's basename. Yields
// `<src-basename>_mock.go` next to the source by default;
// project-level / per-plugin output config and CLI overrides
// can reshape the destination per run.
const FilenameSuffix = "_mock.go"

// ErrDirective is the sentinel wrapped by the plugin's runtime
// directive-validation diagnostics. Tests use [errors.Is] to
// assert on the family without depending on the prose.
var ErrDirective = errors.New("mock: directive")

// Options carries the plugin's user-tunable settings.
type Options struct {
	// Suffix is appended to the source interface name to form the
	// mock struct's identifier — `Store` → `StoreMock`.
	Suffix string `eidos:"suffix,default=Mock"`

	// FieldPrefix is prepended to each method name for the
	// func-valued override field — `Get` → `OnGet`. Choose
	// `Func` as a suffix style via [Options.FieldSuffix] when an
	// existing codebase already uses that convention.
	FieldPrefix string `eidos:"field_prefix,default=On"`

	// FieldSuffix is an alternative to [Options.FieldPrefix] —
	// when non-empty, the field becomes `<Method><FieldSuffix>`
	// instead of `<FieldPrefix><Method>`. Empty by default; the
	// plugin uses FieldPrefix unless FieldSuffix is explicitly
	// configured.
	FieldSuffix string `eidos:"field_suffix"`
}

// Plugin is the production mock generator. Construct via [New];
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

// Priority places the plugin in the composition bucket — it runs
// after foundation generators (e.g. a repo generator) and before
// cross-cutting weavers that decorate its output.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorComposition }

// Provides advertises the `mock` capability so dependent plugins
// can declare a `Requires` edge.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — the plugin has no upstream dependency.
func (*Plugin) Requires() []string { return nil }

// FilenameSuffix returns the per-source-interface suffix the
// routing layer appends. Returns the empty string for any
// language other than Go.
func (*Plugin) FilenameSuffix(lang string) string {
	if lang == Language {
		return FilenameSuffix
	}
	return ""
}

// Directives declares the `+gen:mock` schema. The framework's
// directive registry rejects duplicate registrations at Build
// time so two mock plugins in one pipeline surface as a hard
// error rather than silent shadowing.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			On(node.KindInterface).
			Describe("Opts the host interface into mock generation.").
			Build(),
	}
}

// Generate walks every source-side interface carrying `+gen:mock`
// and emits the corresponding mock struct. Each interface yields a
// per-interface [builder.PackageBuilder] anchored on the source
// interface; the store merges packages sharing a path so multiple
// mocks under the same source package compose into one emit
// bucket without explicit grouping at this site.
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
	for _, iface := range ctx.Reader.Interfaces().Slice() {
		if !iface.HasPositiveDirective(DirectiveName) {
			continue
		}
		pkg := builder.For(Name).Anchor(iface)
		p.emitMock(pkg, iface)
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

// fieldNameFor returns the func-valued field name the mock uses
// to back an interface method named `method`. Honours the
// [Options.FieldSuffix] / [Options.FieldPrefix] split.
func (p *Plugin) fieldNameFor(method string) string {
	if p.opts.FieldSuffix != "" {
		return method + p.opts.FieldSuffix
	}
	return p.opts.FieldPrefix + method
}
