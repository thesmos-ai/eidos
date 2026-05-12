// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package auditweaver appends an `audit.Record` call to the
// [emit.Method.Prebody] slot of every method in the emit store —
// a cross-cutting concern paired with the debug-weaver entry
// trace. The plugin runs in [priority.GeneratorCrossCutting] and
// declares `Requires: ["trace"]` so plan resolution orders it after
// debug-weaver; the rendered prebody therefore lists the debug
// trace first and the audit record second.
//
// `-gen:audit` on an emit method skips the contribution.
package auditweaver

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
)

// Name is the plugin's stable identifier.
const Name = "audit-weaver"

// Capability is the capability label this plugin advertises.
const Capability = "audit"

// RequiresTrace names the upstream capability this plugin depends
// on so the plan orders the trace contributor (typically
// debug-weaver) first.
const RequiresTrace = "trace"

// DirectiveName is the bare directive name read from emit methods
// to suppress the audit contribution on a per-method basis.
const DirectiveName directive.Name = "audit"

// EntryID is the [emit.Provenance.ID] stamped on every audit-weaver
// prebody contribution. Other cross-cutting plugins may position
// themselves relative to the audit record through
// `builder.Before` / `builder.After`.
const EntryID = "audit.record"

// Plugin is the cross-cutting audit-weaver. The zero value is
// usable — the plugin has no options.
type Plugin struct{}

// New returns a fresh plugin instance.
func New() *Plugin { return &Plugin{} }

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Priority places the plugin in the cross-cutting bucket.
func (*Plugin) Priority() priority.Priority { return priority.GeneratorCrossCutting }

// Provides advertises the audit capability.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires declares the trace capability dependency so the plan
// orders the trace contributor first.
func (*Plugin) Requires() []string { return []string{RequiresTrace} }

// Directives declares the `-gen:audit` schema. The positive form
// is allowed by the framework default but carries no plugin
// semantics — audit-weaver applies to every method unconditionally
// unless suppressed.
func (*Plugin) Directives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(DirectiveName).
			On(node.KindMethod).
			Describe("Suppresses (-) the audit record on the host method.").
			Build(),
	}
}

// Generate walks every emit-store method and appends an
// `audit.Record(<owner>.<method>)` call to its Prebody slot,
// skipping methods that carry `-gen:audit`.
func (*Plugin) Generate(ctx *plugin.GeneratorContext) error {
	c := builder.For(Name, emit.Target{})
	for _, m := range ctx.Reader.EmitMethods().Slice() {
		if m.HasNegatedDirective(DirectiveName) {
			continue
		}
		stmt := emit.NewExprStmt(emit.NewCall(
			emit.NewField(emit.NewIdent("audit"), "Record"),
			emit.NewLiteralString(ownerName(m)+"."+m.Name),
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
// the rendered audit record reads `<Type>.<Method>`. The framework
// guarantees emit-store methods carry either *emit.Struct or
// *emit.Interface as their Owner, so the function does not guard
// against other kinds.
func ownerName(m *emit.Method) string {
	if s, ok := m.Owner.(*emit.Struct); ok {
		return s.Name
	}
	return m.Owner.(*emit.Interface).Name
}
