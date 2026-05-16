// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"
)

// Name is the plugin's stable identifier — surfaced in
// diagnostics, cache-key composition, and capability ordering.
const Name = "builder"

// Capability is the capability label this plugin advertises so
// composition-bucket generators can declare a
// `Requires(builder.Capability)` edge to order themselves after
// the builder is emitted.
const Capability = "builder"

// DirectiveName is the bare `+gen:` directive name read from
// source structs. Positive form opts a struct into builder
// generation; negative form suppresses it.
const DirectiveName sdk.DirectiveName = "builder"

// Language is the target-language identifier whose template
// surface the plugin contributes to. Builders are Go-only today;
// other-language equivalents register their own variants.
const Language = "golang"

// FilenameSuffix is the per-source-struct filename suffix the
// routing layer appends to the source file's basename. Yields
// `<src-basename>_builder.go` next to the source by default; the
// framework's routing layer (project config, `+gen:out`, CLI)
// composes the final placement on top.
const FilenameSuffix = "_builder.go"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// Suffix is appended to the source struct's name to form the
	// emitted builder's identifier — `User` → `UserBuilder`.
	Suffix string `eidos:"suffix,default=Builder"`

	// SetterPrefix is prepended to each exported field name to
	// form the fluent setter method — `Name` → `WithName`. Empty
	// disables the prefix so setters become bare field names —
	// `User.WithName(x)` becomes `User.Name(x)`.
	SetterPrefix string `eidos:"setter_prefix,default=With"`
}

// Plugin is the production builder generator. Construct via [New];
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

// Priority places the plugin in the foundation bucket — it emits
// stand-alone scaffolding other generators may compose against
// (a mock could pair with a builder for richer test data).
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorFoundation }

// Provides advertises the `builder` capability so dependent
// plugins can declare a `Requires` edge.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — the plugin has no upstream dependency.
func (*Plugin) Requires() []string { return nil }

// FilenameSuffix returns the per-source-struct suffix the routing
// layer appends. Returns the empty string for any language other
// than Go.
func (*Plugin) FilenameSuffix(lang string) string {
	if lang == Language {
		return FilenameSuffix
	}
	return ""
}

// Directives declares the `+gen:builder` schema. The framework's
// directive registry rejects duplicate registrations at Build
// time so two builder plugins in one pipeline surface as a hard
// error rather than silent shadowing.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			On(node.KindStruct).
			Describe("Opts the host struct into fluent-builder generation.").
			Build(),
	}
}

// Generate walks every source-side struct carrying `+gen:builder`
// and emits the corresponding builder. Each struct yields a
// per-struct [builder.PackageBuilder] anchored on the source
// struct; the store merges packages sharing a path so multiple
// builders under the same source package compose into one emit
// bucket.
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
	for _, s := range ctx.Reader.Structs().Slice() {
		if !s.HasPositiveDirective(DirectiveName) {
			continue
		}
		pkg := builder.For(Name).Anchor(s)
		p.emitBuilder(pkg, s)
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

// setterName returns the fluent-setter method name for an
// exported field — typically `<SetterPrefix><FieldName>`, or just
// `<FieldName>` when the prefix option is empty.
func (p *Plugin) setterName(fieldName string) string {
	return p.opts.SetterPrefix + fieldName
}
