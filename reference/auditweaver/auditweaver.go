// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package auditweaver appends an audit-record call to the
// [emit.Method.Prebody] slot of every method in the emit store —
// a cross-cutting concern paired with the debug-weaver entry
// trace. The plugin runs in [priority.GeneratorCrossCutting] and
// declares `Requires: ["trace"]` so plan resolution orders it after
// debug-weaver; the rendered prebody therefore lists the debug
// trace first and the audit record second.
//
// `-gen:audit` on an emit method skips the contribution.
//
// # Configurability
//
// [Options.Package] selects the import path of the audit package
// the rendered call references, and [Options.Func] selects the
// function on that package. The renderer registers the import on
// the host file's import set via [emit.NewExternal] — the same
// flow [emit.External] type references use — so the rendered output
// is structurally correct without any plugin-side import-management
// scaffolding.
//
// The defaults target stdlib `log.Printf` so projects without a
// dedicated audit package generate compilable output out of the
// box; production deployments override Package + Func to point at
// their real audit surface.
package auditweaver

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
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

// DefaultPackage is the import path the rendered call resolves to
// when [Options.Package] is unset. Stdlib `log` is the default so
// out-of-the-box generation lands compilable code; the message
// rendered through `log.Printf` is the same fully-qualified
// "<Type>.<Method>" string a dedicated audit package would receive.
const DefaultPackage = "log"

// DefaultFunc is the function name the rendered call resolves to
// when [Options.Func] is unset. Stdlib `log.Printf` accepts the
// `"audit: %s"` format with the method-name argument.
const DefaultFunc = "Printf"

// DefaultFormat is the format string passed as the first call
// argument when [Options.Format] is unset — the printf-style
// template the configured Func receives. `%s` interpolates the
// fully-qualified method name. Set explicitly when targeting an
// audit Func with a different signature.
const DefaultFormat = "audit: %s"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// Package is the import path of the audit package the rendered
	// call references. The renderer registers the import on the
	// host file's import set automatically. Defaults to
	// [DefaultPackage].
	Package string `eidos:"package,default=log"`

	// Func is the function name called on the audit package.
	// Defaults to [DefaultFunc].
	Func string `eidos:"func,default=Printf"`

	// Format is the printf-style first argument to the audit call.
	// `%s` interpolates the fully-qualified method name
	// (`<Type>.<Method>`). Defaults to [DefaultFormat].
	Format string `eidos:"format,default=audit: %s"`
}

// Plugin is the cross-cutting audit-weaver.
type Plugin struct {
	*opt.Holder[Options]
	opts Options
}

// New returns a fresh plugin instance with the options holder
// bound.
func New() *Plugin {
	p := &Plugin{}
	p.Holder = opt.Bind(&p.opts)
	return p
}

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

// Generate walks every emit-store method and appends an audit call
// to its Prebody slot, skipping methods that carry `-gen:audit`.
// The call resolves to `<Options.Package>.<Options.Func>(<format>,
// "<Type>.<Method>")` — the renderer registers the import for
// Options.Package on the host file's import set via the
// [emit.NewExternal] expression.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	c := builder.For(Name, emit.Target{})
	for _, m := range ctx.Reader.EmitMethods().Slice() {
		if m.HasNegatedDirective(DirectiveName) {
			continue
		}
		stmt := emit.NewExprStmt(emit.NewCall(
			emit.NewExternal(p.pkg(), p.funcName()),
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
// the documented default when the option is empty. Centralised so
// the rendered behaviour is consistent across every Generate path
// even if a caller bypasses SetOptions.
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
