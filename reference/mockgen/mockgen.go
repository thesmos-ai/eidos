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
//
// # Output layout
//
// mockgen supports two layouts:
//
//   - Alongside source (default, when [Options.OutputPackage] is unset):
//     one `<src>_mock.go` per targeted interface. Source-side mocks
//     drop next to the source file; emit-side mocks drop next to
//     the upstream-emitted interface's source attribution (or its
//     explicit [emit.Target] when set).
//   - Centralised (when [Options.OutputPackage] is set): every
//     emitted decl lands in one Go package + directory, with
//     filename `<src>.go` for composition with other generators.
package mockgen

import (
	"errors"
	"path/filepath"
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

// FilenameSuffix is appended to the lower-cased target interface
// name to form the alongside-source output filename: `<src>_mock.go`.
const FilenameSuffix = "_mock.go"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// OutputPackage is the Go package name + directory the emitted
	// mock decls land in. When unset (the default), mockgen drops
	// one `<src>_mock.go` file alongside each source / emit-store
	// interface; the router resolves [emit.Target.Dir] from the
	// originating source. Set OutputPackage when the project keeps
	// generated code in a dedicated sibling package.
	OutputPackage string `eidos:"output_package"`

	// Suffix is appended to the targeted interface's name to form
	// the emitted mock struct's identifier (`<Type><Suffix>`).
	// Defaults to `Mock`.
	Suffix string `eidos:"suffix,default=Mock"`
}

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
//
// Output layout depends on [Options.OutputPackage]:
//
//   - Unset (default): one emit.Package per source package, emitted
//     alongside the source files. Emit-side interfaces group by the
//     directory portion of their upstream Target.
//   - Set: every decl lands in a single centralised emit.Package.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	if p.opts.OutputPackage != "" {
		return p.generateCentralised(ctx)
	}
	return p.generateAlongsideSource(ctx)
}

// generateCentralised drops every emitted mock into one shared
// emit.Package keyed by [Options.OutputPackage]. Emit-side mocks
// inherit the upstream interface's Target so they compose into the
// same file as their parent. Source-side mocks use the canonical
// `strings.ToLower(<src>).go` filename, matching repogen / buildergen
// so all generators targeting one source struct land in one file.
func (p *Plugin) generateCentralised(ctx *plugin.GeneratorContext) error {
	c := builder.For(Name, emit.Target{})
	pkg := c.Package(p.opts.OutputPackage, Name+":"+p.opts.OutputPackage)
	for _, ei := range ctx.Reader.EmitInterfaces().Slice() {
		if ei.HasNegatedDirective(DirectiveName) {
			continue
		}
		p.emitForEmitInterface(pkg, ei, ei.Target)
	}
	for _, si := range ctx.Reader.Interfaces().Slice() {
		if !si.HasPositiveDirective(DirectiveName) {
			continue
		}
		p.emitForSourceInterface(pkg, si, p.centralisedTargetFor(si.Name), emit.External(si.Package, si.Name))
	}
	out, err := pkg.Build()
	if err != nil {
		return err
	}
	if err := ctx.Store.Emit().AddPackage(out); err != nil {
		return errors.Join(errAddPackage, err)
	}
	return nil
}

// generateAlongsideSource emits one mock per targeted interface with
// Target.Dir left empty (router fills) and filename `<src>_mock.go`.
// Source interfaces group by their owning source package; emit-side
// interfaces are emitted into a per-plugin synthetic package whose
// name comes from each interface's upstream Target.Package.
func (p *Plugin) generateAlongsideSource(ctx *plugin.GeneratorContext) error {
	var firstErr error
	if err := p.emitSourceMocksPerPackage(ctx); err != nil {
		firstErr = err
	}
	if err := p.emitInterfaceMocks(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// emitSourceMocksPerPackage walks the node store grouped by source
// package and emits one mock per `+gen:mock` interface, keyed by the
// source package so the rendered file's package clause matches. The
// emitted target carries the source package's import path so the
// renderer elides same-package qualifiers — the plugin can use
// [emit.External] uniformly for the mocked interface reference and
// the renderer drops the self-import.
func (p *Plugin) emitSourceMocksPerPackage(ctx *plugin.GeneratorContext) error {
	var firstErr error
	ctx.Reader.Packages().Each(func(srcPkg *node.Package) {
		matches := matchingInterfaces(srcPkg)
		if len(matches) == 0 {
			return
		}
		c := builder.For(Name, emit.Target{})
		pkg := c.Package(srcPkg.Name, Name+":src:"+srcPkg.Path)
		for _, si := range matches {
			target := emit.Target{
				Filename:   strings.ToLower(si.Name) + FilenameSuffix,
				Package:    srcPkg.Name,
				ImportPath: srcPkg.Path,
			}
			p.emitForSourceInterface(pkg, si, target, emit.External(si.Package, si.Name))
		}
		out, err := pkg.Build()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return
		}
		if err := ctx.Store.Emit().AddPackage(out); err != nil {
			if firstErr == nil {
				firstErr = errors.Join(errAddPackage, err)
			}
		}
	})
	return firstErr
}

// emitInterfaceMocks emits one mock per upstream emit interface that
// hasn't opted out via `-gen:mock`. Mocks group by the upstream
// interface's Target so each rendered output lands in the same
// directory as the interface it shadows.
func (p *Plugin) emitInterfaceMocks(ctx *plugin.GeneratorContext) error {
	groups := map[string][]*emit.Interface{}
	keys := []string{}
	for _, ei := range ctx.Reader.EmitInterfaces().Slice() {
		if ei.HasNegatedDirective(DirectiveName) {
			continue
		}
		key := emitInterfaceGroupKey(ei)
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], ei)
	}
	var firstErr error
	for _, key := range keys {
		ifaces := groups[key]
		pkgName := ifaces[0].Target.Package
		c := builder.For(Name, emit.Target{})
		pkg := c.Package(pkgName, Name+":emit:"+key)
		for _, ei := range ifaces {
			target := emit.Target{
				Dir:        ei.Target.Dir,
				Filename:   strings.ToLower(ei.Name) + FilenameSuffix,
				Package:    ei.Target.Package,
				ImportPath: ei.Target.ImportPath,
			}
			p.emitForEmitInterface(pkg, ei, target)
		}
		out, err := pkg.Build()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := ctx.Store.Emit().AddPackage(out); err != nil {
			if firstErr == nil {
				firstErr = errors.Join(errAddPackage, err)
			}
		}
	}
	return firstErr
}

// emitInterfaceGroupKey returns the bucketing key for an upstream
// emit interface: the directory portion of its target plus the
// package name. Ensures interfaces destined for the same rendered
// directory share one emit.Package.
func emitInterfaceGroupKey(i *emit.Interface) string {
	return filepath.Clean(i.Target.Dir) + "#" + i.Target.Package
}

// errAddPackage is the sentinel wrapped around store-side AddPackage
// failures. Tests and callers detect the class with errors.Is.
//
//nolint:gochecknoglobals // sentinel.
var errAddPackage = errors.New("mockgen: add package to store")

// matchingInterfaces returns srcPkg's source-side interfaces opted
// into mocking by `+gen:mock`.
func matchingInterfaces(srcPkg *node.Package) []*node.Interface {
	out := make([]*node.Interface, 0, len(srcPkg.Interfaces))
	for _, i := range srcPkg.Interfaces {
		if i.HasPositiveDirective(DirectiveName) {
			out = append(out, i)
		}
	}
	return out
}

// centralisedTargetFor builds the centralised-layout Target for a
// mock named after srcName. Filename matches the convention repogen
// and buildergen use so multiple plugin outputs targeting the same
// source struct compose into one file.
func (p *Plugin) centralisedTargetFor(srcName string) emit.Target {
	return emit.Target{
		Dir:      p.opts.OutputPackage,
		Filename: strings.ToLower(srcName) + ".go",
		Package:  p.opts.OutputPackage,
	}
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
// interface using the supplied Target. The Mock references the
// source interface by [emit.Internal] so the rendered struct
// correctly resolves the in-target name regardless of the emit
// interface's own package.
func (p *Plugin) emitForEmitInterface(pkg *builder.PackageBuilder, i *emit.Interface, target emit.Target) {
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
	p.emitMock(pkg, i.Name, emit.Internal(i), target, i.Origin(), sigs)
}

// emitForSourceInterface emits a Mock struct for a source-side
// interface, lifting its node-layer types into emit refs through
// [refconv.FromNode] so the generated mock parses against the same
// signatures the source declares. ifaceRef is the reference the
// emitted mock uses for the source interface (passed by the caller
// so centralised layout gets [emit.External] while alongside-source
// layout gets [emit.Builtin], avoiding the self-import).
func (p *Plugin) emitForSourceInterface(
	pkg *builder.PackageBuilder,
	i *node.Interface,
	target emit.Target,
	ifaceRef emit.Ref,
) {
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
	p.emitMock(pkg, i.Name, ifaceRef, target, i, sigs)
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
	origin node.Node,
	sigs []methodSig,
) {
	_ = ifaceRef // reserved for future "var _ Iface = (*Mock)(nil)" emission.
	mockName := ifaceName + p.opts.Suffix

	pkg.Struct(mockName, func(b *builder.StructBuilder) {
		b.Target(target)
		b.Node().OriginNode = origin
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
