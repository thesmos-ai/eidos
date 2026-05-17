// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package debugweaver appends a debug-trace call to the
// [emit.Method.Prebody] slot of every method in the emit store —
// the canonical "entry trace" cross-cutting concern. The plugin
// runs in [sdk.GeneratorCrossCutting] and advertises the
// `trace` capability so other cross-cutting plugins (audit,
// metric, …) can declare a Requires dependency on a known
// trace-entry contribution.
//
// `-gen:debug` on an emit method skips that method; methods
// without the directive get the contribution. Each appended
// statement carries [Provenance.ID] `trace.entry` so later
// cross-cutting plugins can position themselves relative to the
// debug entry trace through `builder.Before` / `builder.After`.
//
// # Configurability
//
// [Options.Package] selects the import path of the package the
// rendered call references; [Options.Func] selects the function on
// that package. The renderer registers the import on the host
// file's import set via [emit.NewExternal] — the same flow
// [emit.External] type references use — so the rendered output is
// structurally correct without any plugin-side import-management
// scaffolding.
package debugweaver

import (
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"
)

// Name is the plugin's stable identifier.
const Name = "debug-weaver"

// Capability is the capability label this plugin advertises so
// downstream cross-cutting contributors (audit, metric, …) can
// declare ordering through `Requires`.
const Capability = "trace"

// DirectiveName is the bare directive name the plugin reads from
// emit methods to suppress its contribution on a per-method basis.
const DirectiveName sdk.DirectiveName = "debug"

// EntryID is the [emit.Provenance.ID] stamped on every debug-weaver
// prebody contribution. Cross-cutting plugins that want to position
// their own statement relative to the entry trace pass this id to
// [builder.Before] / [builder.After].
const EntryID = "trace.entry"

// DefaultPackage is the import path the rendered call resolves to
// when [Options.Package] is unset. Stdlib `log` keeps the demo
// self-contained — projects override Package + Func to point at
// their real trace surface.
const DefaultPackage = "log"

// DefaultFunc is the function name the rendered call resolves to
// when [Options.Func] is unset.
const DefaultFunc = "Printf"

// DefaultFormat is the printf-style first argument to the trace
// call when [Options.Format] is unset. `%s` interpolates the
// fully-qualified method name (`<Type>.<Method>`).
const DefaultFormat = "debug: %s entered"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// Package is the import path of the package the rendered trace
	// call references. Defaults to [DefaultPackage].
	Package string `eidos:"package,default=log"`

	// Func is the function name called on the trace package.
	// Defaults to [DefaultFunc].
	Func string `eidos:"func,default=Printf"`

	// Format is the printf-style first argument to the trace call.
	// Defaults to [DefaultFormat].
	Format string `eidos:"format,default=debug: %s entered"`
}

// Plugin is the cross-cutting debug-weaver.
type Plugin struct {
	*sdk.Holder[Options]
	opts Options
}

// New returns a fresh plugin instance with the options holder
// bound.
func New() *Plugin {
	p := &Plugin{}
	p.Holder = sdk.BindOptions(&p.opts)
	return p
}

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Priority places the plugin in the cross-cutting bucket so it
// runs after foundation and composition generators.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorCrossCutting }

// Provides advertises the trace capability.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — debug-weaver has no upstream dependency.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `-gen:debug` schema. The positive form
// is allowed by the framework default but carries no plugin
// semantics — debug-weaver applies to every method unconditionally
// unless suppressed.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			On(node.KindMethod).
			Describe("Suppresses (-) the debug-entry trace on the host method.").
			Build(),
	}
}

// Generate walks every emit-store method and appends a trace call
// to its Prebody slot, except for methods that carry `-gen:debug`.
// The call resolves to `<Options.Package>.<Options.Func>(<format>,
// "<Type>.<Method>")` — the renderer registers the import for
// Options.Package on the host file's import set via the
// [emit.NewExternal] expression.
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name, sdk.EmitTarget{})
	for _, m := range ctx.Reader.EmitMethods().Slice() {
		if m.HasNegatedDirective(DirectiveName) {
			continue
		}
		stmt := emit.NewExprStmt(emit.NewCall(
			sdk.NewExternal(p.pkg(), p.funcName()),
			emit.NewLiteralString(p.format()),
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

// pkg / funcName / format return the configured option value or
// the documented default when the option is empty.
func (p *Plugin) pkg() string {
	if p.opts.Package != "" {
		return p.opts.Package
	}
	return DefaultPackage
}

func (p *Plugin) funcName() string {
	if p.opts.Func != "" {
		return p.opts.Func
	}
	return DefaultFunc
}

func (p *Plugin) format() string {
	if p.opts.Format != "" {
		return p.opts.Format
	}
	return DefaultFormat
}

// ownerName returns the simple receiver-type name of m's owner so
// the rendered log message reads `<Type>.<Method>` — the common
// "trace this method on this struct" mental model. Methods on
// interfaces, methods on structs, and methods on source-side
// types are all handled uniformly via [contract.Owner.OwnerName],
// so the function never type-switches the underlying Owner kind.
func ownerName(m *emit.Method) string {
	return m.OwnerName()
}
