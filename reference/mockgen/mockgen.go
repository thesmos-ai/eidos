// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package mockgen synthesises a mock implementation for every
// targeted interface. A targeted interface is either an emit-store
// interface produced by an upstream generator (typically repogen's
// `<Type>Repository`) or a source-side interface carrying
// `+gen:mock`.
//
// For each target, the plugin emits a `<Type><Suffix>` struct with
// one func-valued field per method (`<Method>Func`) plus an
// implementing method that dispatches to that field. The emitted
// mock is suitable for table-driven tests without an extra mocking
// dependency.
//
// mockgen sits in the [priority.GeneratorComposition] bucket; the
// `Requires: ["repository"]` declaration documents the dependency
// on repogen's output even though the strict-by-priority bucket
// ordering already runs foundation generators first.
package mockgen

import (
	"errors"
	"strconv"
	"strings"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
	"go.thesmos.sh/eidos/reference/internal/refconv"
)

// Name is the plugin's stable identifier surfaced through
// [plugin.Plugin.Name].
const Name = "mockgen"

// Capability is the capability label mockgen advertises through
// [plugin.CapabilityProvider.Provides].
const Capability = "mock"

// RequiresRepository names the upstream capability mockgen documents
// a dependency on so [pipeline.Pipeline.Plan] records the
// cross-bucket relationship.
const RequiresRepository = "repository"

// DirectiveName is the bare directive name (without the `+gen:` or
// `-gen:` prefix) the plugin reads from interfaces on both the
// source and the emit side.
const DirectiveName directive.Name = "mock"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// OutputPackage is the Go package name + directory the emitted
	// mock decls land in. Required.
	OutputPackage string `eidos:"output_package,required"`

	// Suffix is appended to the targeted interface's name to form
	// the emitted mock struct's identifier (`<Type><Suffix>`).
	// Defaults to `Mock`.
	Suffix string `eidos:"suffix,default=Mock"`
}

// errMissingOutputPackage is returned when [Plugin.Generate] runs
// without the required OutputPackage option populated.
//
//nolint:gochecknoglobals // sentinel.
var errMissingOutputPackage = errors.New("mockgen: OutputPackage option is required")

// Plugin is the mock-implementation generator. The zero value is
// unusable — go through [New] so the embedded [opt.Holder] binds
// to the plugin's options field.
type Plugin struct {
	*opt.Holder[Options]
	opts Options
}

// New returns a fresh plugin instance with the options holder bound.
// The pipeline overlays caller-supplied option values via
// [Plugin.SetOptions] (promoted from [opt.Holder]) at Build time.
func New() *Plugin {
	p := &Plugin{}
	p.Holder = opt.Bind(&p.opts)
	return p
}

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Priority places the plugin in the composition generator bucket so
// it runs after the foundation generators that synthesise the
// interfaces it mocks.
func (*Plugin) Priority() priority.Priority { return priority.GeneratorComposition }

// Provides returns [Capability].
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns the repository capability label. The strict
// cross-bucket ordering already runs foundation before composition,
// but the declared dependency carries documentary intent.
func (*Plugin) Requires() []string { return []string{RequiresRepository} }

// Directives declares the `+gen:mock` / `-gen:mock` schema. Positive
// directives opt source-side interfaces in; negated directives skip
// emit-side interfaces that would otherwise be mocked.
func (*Plugin) Directives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(DirectiveName).
			On(node.KindInterface).
			On(directive.Kind("emit.interface")).
			Describe("Opts a source interface into mock generation (+) or skips an emit interface (-).").
			Build(),
	}
}

// Generate emits a mock struct for every targeted interface. The
// plugin walks two sources: emit-store interfaces (where foundation
// generators land their output) and source interfaces carrying
// `+gen:mock`. Emit-side interfaces are mocked unless they carry
// `-gen:mock`; source-side interfaces are mocked only when they
// carry `+gen:mock`.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	if p.opts.OutputPackage == "" {
		return errMissingOutputPackage
	}
	c := builder.For(Name, emit.Target{})
	pkg := c.Package(p.opts.OutputPackage, p.opts.OutputPackage)
	for _, ei := range ctx.Reader.EmitInterfaces().Slice() {
		if ei.HasNegatedDirective(DirectiveName) {
			continue
		}
		p.emitForEmitInterface(pkg, ei)
	}
	for _, si := range ctx.Reader.Interfaces().Slice() {
		if !si.HasPositiveDirective(DirectiveName) {
			continue
		}
		p.emitForSourceInterface(pkg, si)
	}
	out, err := pkg.Build()
	if err != nil {
		return err
	}
	return ctx.Store.Emit().AddPackage(out)
}

// methodSig is the minimal per-method information mockgen needs to
// emit one Mock-struct method. Both the emit-interface and the
// source-interface entry points lower their respective method shapes
// onto this common form.
type methodSig struct {
	name    string
	params  []paramSig
	returns []emit.Ref
}

// paramSig describes one positional parameter — name and type.
// Anonymous parameters carry an empty name; the emit code rewrites
// them to `arg<N>` so the mock body can reference them.
type paramSig struct {
	name string
	typ  emit.Ref
}

// emitForEmitInterface emits a Mock struct for an emit-store
// interface. The Mock references the source interface by
// [emit.Internal] so the rendered struct correctly resolves the
// in-target name regardless of the emit interface's own package.
func (p *Plugin) emitForEmitInterface(pkg *builder.PackageBuilder, i *emit.Interface) {
	sigs := make([]methodSig, 0, len(i.Methods))
	for _, m := range i.Methods {
		params := make([]paramSig, 0, len(m.Params))
		for _, mp := range m.Params {
			params = append(params, paramSig{name: mp.Name, typ: mp.Type})
		}
		returns := make([]emit.Ref, 0, len(m.Returns))
		for _, r := range m.Returns {
			returns = append(returns, r.Type)
		}
		sigs = append(sigs, methodSig{name: m.Name, params: params, returns: returns})
	}
	p.emitMock(pkg, i.Name, emit.Internal(i), i.Target, sigs)
}

// emitForSourceInterface emits a Mock struct for a source-side
// interface, lifting its node-layer types into emit refs through
// [refconv.FromNode] so the generated mock parses against the same
// signatures the source declares.
func (p *Plugin) emitForSourceInterface(pkg *builder.PackageBuilder, i *node.Interface) {
	sigs := make([]methodSig, 0, len(i.Methods))
	for _, m := range i.Methods {
		params := make([]paramSig, 0, len(m.Params))
		for _, mp := range m.Params {
			params = append(params, paramSig{name: mp.Name, typ: refconv.FromNode(mp.Type)})
		}
		returns := make([]emit.Ref, 0, len(m.Returns))
		for _, r := range m.Returns {
			returns = append(returns, refconv.FromNode(r))
		}
		sigs = append(sigs, methodSig{name: m.Name, params: params, returns: returns})
	}
	p.emitMock(pkg, i.Name, emit.External(i.Package, i.Name), p.targetFor(i.Name), sigs)
}

// emitMock appends one Mock struct decl carrying the func-valued
// fields and dispatching methods for every signature in sigs. The
// mocked interface is rendered through ifaceRef in field-receiver
// position so callers can pass the Mock anywhere the source
// interface is required.
func (p *Plugin) emitMock(
	pkg *builder.PackageBuilder,
	ifaceName string,
	ifaceRef emit.Ref,
	target emit.Target,
	sigs []methodSig,
) {
	_ = ifaceRef // reserved for future "var _ Iface = (*Mock)(nil)" emission.
	mockName := ifaceName + p.opts.Suffix

	pkg.Struct(mockName, func(b *builder.StructBuilder) {
		b.Target(target)
		b.Docs(mockName + " is a func-valued mock implementation of " + ifaceName + ".")
		for _, s := range sigs {
			b.Field(s.name+"Func", funcRefFor(s), nil)
		}
		recv := emit.Ptr(emit.Internal(b.Node()))
		for _, s := range sigs {
			b.Method(s.name, func(m *builder.MethodBuilder) {
				m.Receiver("m", recv)
				for i, param := range s.params {
					m.Param(paramName(param.name, i), param.typ)
				}
				for _, ret := range s.returns {
					m.Return(ret)
				}
				m.Body(dispatchBody(s)...)
			})
		}
	})
}

// targetFor builds the canonical [emit.Target] for the supplied
// source-interface name. Filename matches the convention repogen
// and buildergen use so multiple plugin outputs targeting the same
// source struct land in one file when the OutputPackage matches.
func (p *Plugin) targetFor(srcName string) emit.Target {
	return emit.Target{
		Dir:      p.opts.OutputPackage,
		Filename: strings.ToLower(srcName) + ".go",
		Package:  p.opts.OutputPackage,
	}
}

// funcRefFor builds the `func(<params>) <returns>` type for the
// func-valued field that backs a Mock method. Anonymous parameters
// keep their empty names — fields render in func-type position so
// the names don't appear in source.
func funcRefFor(s methodSig) emit.Ref {
	params := make([]emit.Ref, 0, len(s.params))
	for _, p := range s.params {
		params = append(params, p.typ)
	}
	return emit.FuncOf(params, s.returns)
}

// paramName returns the in-method identifier for parameter index i:
// the source name when non-empty, or `arg<N>` when the source name
// is empty so the dispatch body can reference the parameter.
func paramName(name string, i int) string {
	if name != "" {
		return name
	}
	return "arg" + strconv.Itoa(i)
}

// dispatchBody returns the statement list for one Mock method's
// body: `[return ]m.<Method>Func(<args...>)`. Zero-return methods
// drop the leading return.
func dispatchBody(s methodSig) []*emit.Stmt {
	args := make([]*emit.Expr, 0, len(s.params))
	for i, p := range s.params {
		args = append(args, emit.NewIdent(paramName(p.name, i)))
	}
	call := emit.NewCall(
		emit.NewField(emit.NewIdent("m"), s.name+"Func"),
		args...,
	)
	if len(s.returns) == 0 {
		return []*emit.Stmt{emit.NewExprStmt(call)}
	}
	return []*emit.Stmt{emit.NewReturn(call)}
}
