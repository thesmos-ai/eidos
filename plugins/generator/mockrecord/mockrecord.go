// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mockrecord

import (
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/plugins/generator/mock"
	"go.thesmos.sh/eidos/sdk"
)

// Name is the plugin's stable identifier — surfaced in diagnostics,
// cache-key composition, and capability ordering.
const Name = "mockrecord"

// Capability is the capability label this plugin advertises so
// dependent plugins (fault-injection, replay) can declare a
// `Requires(mockrecord.Capability)` edge to order themselves after
// the recorder's contributions land.
const Capability = "mock-recording"

// Language is the target-language identifier whose surface this
// plugin contributes to. Mock recording is Go-only today.
const Language = "golang"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// FieldSuffix is appended to each method name to form the
	// per-method call-log field on the mock — `Get` → `GetCalls`.
	FieldSuffix string `eidos:"field_suffix,default=Calls"`

	// CallStructSuffix is appended to `<MockName><Method>` to form
	// the per-method call-record struct identifier — Searcher's
	// `Get` method yields `SearcherMockGetCall` under the default.
	CallStructSuffix string `eidos:"call_struct_suffix,default=Call"`
}

// Plugin is the production mock-recording cross-cutter. Construct
// via [New]; the zero value is unusable because the options holder
// is nil.
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
// decorates the mock plugin's output via slot contributions and
// must run after the mock has populated the emit store.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorCrossCutting }

// Provides advertises the `mock-recording` capability so dependent
// plugins (replay generators, assertion helpers) can declare a
// `Requires` edge to order themselves after the recorder.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns the mock capability — the plugin emits against
// the mock plugin's output and must run after it.
func (*Plugin) Requires() []string { return []string{mock.Capability} }

// Generate walks every emit struct stamped with [mock.MetaIface]
// and decorates it with per-method call-recording machinery: a
// `<MockName><Method>Call` typed-param struct, a slice field
// appended via [emit.Struct.FieldsSlot], and a per-method
// Prebody contribution that records the call before dispatch.
// The per-mock work is in [Plugin.emitRecording]; this method
// owns the gate, per-target package wiring, and store update.
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
	for _, s := range ctx.Reader.EmitStructs().Slice() {
		if _, ok := mock.MetaIface.Get(s.Meta()); !ok {
			continue
		}
		pkg := builder.For(Name).Anchor(s.Origin())
		if err := p.emitRecording(pkg, s); err != nil {
			return err
		}
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
