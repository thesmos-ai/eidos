// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package debugweaver appends a `log.Printf` call to the
// [emit.Method.Prebody] slot of every method in the emit store —
// the canonical "entry trace" cross-cutting concern. The plugin
// runs in [priority.GeneratorCrossCutting] and advertises the
// `trace` capability so other cross-cutting plugins (audit,
// metric, …) can declare a Requires dependency on a known
// trace-entry contribution.
//
// `-gen:debug` on an emit method skips that method; methods
// without the directive get the contribution. Each appended
// statement carries [Provenance.ID] `trace.entry` so later
// cross-cutting plugins can position themselves relative to the
// debug entry trace through `builder.Before` / `builder.After`.
package debugweaver

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
)

// Name is the plugin's stable identifier.
const Name = "debug-weaver"

// Capability is the capability label this plugin advertises so
// downstream cross-cutting contributors (audit, metric, …) can
// declare ordering through `Requires`.
const Capability = "trace"

// DirectiveName is the bare directive name the plugin reads from
// emit methods to suppress its contribution on a per-method basis.
const DirectiveName directive.Name = "debug"

// EntryID is the [emit.Provenance.ID] stamped on every debug-weaver
// prebody contribution. Cross-cutting plugins that want to position
// their own statement relative to the entry trace pass this id to
// [builder.Before] / [builder.After].
const EntryID = "trace.entry"

// Plugin is the cross-cutting debug-weaver. The zero value is
// usable — the plugin has no options.
type Plugin struct{}

// New returns a fresh plugin instance.
func New() *Plugin { return &Plugin{} }

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Priority places the plugin in the cross-cutting bucket so it
// runs after foundation and composition generators.
func (*Plugin) Priority() priority.Priority { return priority.GeneratorCrossCutting }

// Provides advertises the trace capability.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — debug-weaver has no upstream dependency.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `-gen:debug` schema. The positive form
// is allowed by the framework default but carries no plugin
// semantics — debug-weaver applies to every method unconditionally
// unless suppressed.
func (*Plugin) Directives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(DirectiveName).
			On(node.KindMethod).
			Describe("Suppresses (-) the debug-entry trace on the host method.").
			Build(),
	}
}

// Generate walks every emit-store method and appends a Printf
// call to its Prebody slot, except for methods that carry
// `-gen:debug`. The Printf message is the fully-qualified method
// name resolved at emit time so renderers see a literal string.
func (*Plugin) Generate(ctx *plugin.GeneratorContext) error {
	c := builder.For(Name, emit.Target{})
	for _, m := range ctx.Reader.EmitMethods().Slice() {
		if m.HasNegatedDirective(DirectiveName) {
			continue
		}
		stmt := emit.NewExprStmt(emit.NewCall(
			emit.NewField(emit.NewIdent("log"), "Printf"),
			emit.NewLiteralString("debug: "+ownerName(m)+"."+m.Name+" entered"),
		))
		// AppendPrebody can only fail when host is nil or carries
		// an unsupported kind — neither possible for the *emit.Method
		// values EmitMethods yields. The Append is therefore
		// infallible at this call site.
		_ = c.AppendPrebody(m, stmt, EntryID)
	}
	return nil
}

// ownerName returns the simple receiver-type name of m's owner so
// the rendered log message reads `<Type>.<Method>` — the common
// "trace this method on this struct" mental model. Methods on
// interfaces and methods on structs are both handled; the
// framework guarantees these two kinds for every method that
// reaches the emit-store methods bucket, so the function does not
// guard against other Owner kinds.
func ownerName(m *emit.Method) string {
	if s, ok := m.Owner.(*emit.Struct); ok {
		return s.Name
	}
	return m.Owner.(*emit.Interface).Name
}
