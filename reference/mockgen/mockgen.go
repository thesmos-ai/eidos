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
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/core/srcfile"
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

// FilenameSuffix is appended to the source-file basename (without
// the `.go` extension) to form the alongside-source output
// filename: `<src-file>_mock.go`. Every `+gen:mock` interface
// declared in `searcher.go` therefore composes into one
// `searcher_mock.go`.
const FilenameSuffix = "_mock.go"

// TestFilenameSuffix is appended to the source-file basename
// (without the `.go` extension) when [Options.Test] is set:
// `<src-file>_test.go`. The Go test toolchain compiles `_test.go`
// files only at test time, so the generated mock stays out of
// production builds.
const TestFilenameSuffix = "_test.go"

// TestPackageSuffix is appended to the source package's short name
// to form the rendered file's package clause when [Options.Test]
// is set: `<srcPkg.Name>_test`. The same suffix is applied to the
// source package's import path to derive a distinct
// [emit.Target.ImportPath] — distinct so same-package elision does
// NOT fire when the test-package mock references types in the
// regular package.
const TestPackageSuffix = "_test"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// OutputPackage is the Go package name + directory the emitted
	// mock decls land in. When unset (the default), mockgen drops
	// one `<src>_mock.go` file alongside each source / emit-store
	// interface; the router resolves [emit.Target.Dir] from the
	// originating source. Set OutputPackage when the project keeps
	// generated code in a dedicated sibling package.
	//
	// OutputPackage is ignored when [Options.Test] is set — test
	// mocks always land alongside the source under the
	// `<srcPkg>_test` package, regardless of any centralised
	// OutputPackage routing.
	OutputPackage string `eidos:"output_package"`

	// Suffix is appended to the targeted interface's name to form
	// the emitted mock struct's identifier (`<Type><Suffix>`).
	// Defaults to `Mock`.
	Suffix string `eidos:"suffix,default=Mock"`

	// Test enables test-package emission. When true, every emitted
	// mock lands in `<src>_test.go` next to the source file under
	// a `<srcPkg>_test` package — the Go-conventional placement
	// for fakes that should compile only at test time. The test
	// package's import identity differs from the regular
	// package's, so references back to the regular package's
	// types qualify rather than elide.
	Test bool `eidos:"test,default=false"`
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

// FilenameSuffix returns [FilenameSuffix] — the per-source filename
// suffix the Layout phase appends to the source's basename when
// composing the rendered output path for every decl this plugin
// emits. Implements [plugin.FilenameProvider]. Test-mode mocks use
// [TestFilenameSuffix] instead; the per-decl suffix override flows
// through emit-meta when [Options.Test] is enabled.
func (*Plugin) FilenameSuffix() string { return FilenameSuffix }

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
// Output layout depends on [Options.Test] / [Options.OutputPackage]:
//
//   - [Options.Test] set: one mock per `+gen:mock` source interface,
//     emitted to `<src>_test.go` next to the source under a
//     `<srcPkg.Name>_test` package. OutputPackage is ignored.
//   - OutputPackage set (and Test unset): every decl lands in a
//     single centralised emit.Package.
//   - Both unset (default): one emit.Package per source package,
//     emitted alongside the source files. Emit-side interfaces
//     group by the directory portion of their upstream Target.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	if p.opts.Test {
		return p.generateTestPackage(ctx)
	}
	if p.opts.OutputPackage != "" {
		return p.generateCentralised(ctx)
	}
	return p.generateAlongsideSource(ctx)
}

// generateTestPackage emits one mock per `+gen:mock` source
// interface into `<src>_test.go` under the source's `_test`
// package. Emit-side interfaces (produced by upstream generators
// like repogen) are skipped — they belong in the regular package's
// fixtures, not the test package; downstream callers that want
// emit-side mocks in the test package can run mockgen twice with
// different configurations.
//
// The synthetic ImportPath (`<srcPkg.Path>_test`) ensures the
// renderer does NOT elide references back into the regular package
// — `storage.Repository[T]` rendered from inside `storage_test`
// resolves through an `import "example.com/.../storage"` rather
// than as a bare `Repository[T]`.
func (p *Plugin) generateTestPackage(ctx *plugin.GeneratorContext) error {
	var firstErr error
	ctx.Reader.Packages().Each(func(srcPkg *node.Package) {
		matches := matchingInterfaces(srcPkg)
		if len(matches) == 0 {
			return
		}
		testPkgName := srcPkg.Name + TestPackageSuffix
		testImportPath := srcPkg.Path + TestPackageSuffix
		c := builder.For(Name, emit.Target{})
		pkg := c.Package(testPkgName, testImportPath)
		for _, si := range matches {
			target := emit.Target{
				Filename:   srcfile.WithSuffix(si.Pos(), si.Name, TestFilenameSuffix),
				Package:    testPkgName,
				ImportPath: testImportPath,
			}
			// External ref into the regular package — the test
			// package's distinct ImportPath keeps the renderer
			// from eliding the qualifier.
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

// generateCentralised drops every emitted mock into one shared
// emit.Package keyed by [Options.OutputPackage]. Emit-side mocks
// inherit the upstream interface's Target so they compose into the
// same file as their parent. Source-side mocks use the canonical
// `strings.ToLower(<src>).go` filename, matching repogen / buildergen
// so all generators targeting one source struct land in one file.
func (p *Plugin) generateCentralised(ctx *plugin.GeneratorContext) error {
	c := builder.For(Name, emit.Target{})
	pkg := c.Package(p.opts.OutputPackage, p.opts.OutputPackage)
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
		pkg := c.Package(srcPkg.Name, srcPkg.Path)
		for _, si := range matches {
			target := emit.Target{
				Filename:   srcfile.WithSuffix(si.Pos(), si.Name, FilenameSuffix),
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
		// Path identity for the emit.Package: the upstream
		// interface's ImportPath when set (canonical), else the
		// grouping key (Dir#Package) so distinct rendered packages
		// stay in distinct emit.Packages. Store-side merging joins
		// multi-plugin contributions into one package on collision.
		path := ifaces[0].Target.ImportPath
		if path == "" {
			path = key
		}
		c := builder.For(Name, emit.Target{})
		pkg := c.Package(pkgName, path)
		for _, ei := range ifaces {
			// Source-file basename comes from the upstream
			// interface's Origin — the source struct (or interface)
			// repogen / a peer generator built the interface from.
			// Falling back to ei.Name keeps emission stable for
			// synthetic interfaces that lack a source attribution.
			pos := emitInterfaceOriginPos(ei)
			target := emit.Target{
				Dir:        ei.Target.Dir,
				Filename:   srcfile.WithSuffix(pos, ei.Name, FilenameSuffix),
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

// emitInterfaceOriginPos returns the source position of the upstream
// emit interface's originating node — typically the source struct
// (or interface) the upstream generator built it from. Returns the
// zero position when no origin is wired; callers fall back to a
// name-derived filename in that case.
func emitInterfaceOriginPos(i *emit.Interface) position.Pos {
	if origin := i.Origin(); origin != nil {
		return origin.Pos()
	}
	return position.Pos{}
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
// interface's own package. Generic emit interfaces propagate their
// type parameters to the mock so the rendered struct, methods, and
// ifaceRef-instantiation all thread `[T1, T2, …]` consistently.
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
	tps := emitTypeParamsFromEmit(i.TypeParams)
	typeArgs := builder.TypeArgsFromEmitParams(i.TypeParams)
	p.emitMock(pkg, i.Name, emit.Internal(i, typeArgs...), target, i.Origin(), tps, sigs)
}

// emitForSourceInterface emits a Mock struct for a source-side
// interface, lifting its node-layer types into emit refs through
// [refconv.FromNode] so the generated mock parses against the same
// signatures the source declares. ifaceRef is the reference the
// emitted mock uses for the source interface (passed by the caller
// so centralised layout gets [emit.External] while alongside-source
// layout gets [emit.External] too — the same-package elision
// performed via [emit.Target.ImportPath] renders it bare).
//
// Generic source interfaces propagate their type parameters to the
// mock and thread the type-arg list through ifaceRef so the
// rendered struct, methods, and embedded reference all carry
// `[T1, T2, …]` consistently.
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
	tps := emitTypeParamsFromNode(i.TypeParams)
	typeArgs := builder.TypeArgsFromNodeParams(i.TypeParams)
	p.emitMock(pkg, i.Name, builder.ApplyTypeArgs(ifaceRef, typeArgs), target, i, tps, sigs)
}

// emitMock appends one Mock struct decl carrying the func-valued
// fields and dispatching methods for every signature in sigs. The
// mocked interface is rendered through ifaceRef in field-receiver
// position so callers can pass the Mock anywhere the source
// interface is required. typeParams (when non-empty) propagate the
// host interface's generic parameters to the mock so the rendered
// struct, methods, and receivers all carry the same `[T1, T2, …]`
// bracket list.
func (p *Plugin) emitMock(
	pkg *builder.PackageBuilder,
	ifaceName string,
	ifaceRef emit.Ref,
	target emit.Target,
	origin node.Node,
	typeParams []emitTypeParamSpec,
	sigs []methodSig,
) {
	_ = ifaceRef // reserved for future "var _ Iface = (*Mock)(nil)" emission.
	mockName := ifaceName + p.opts.Suffix

	pkg.Struct(mockName, func(b *builder.StructBuilder) {
		b.Target(target)
		b.Origin(origin)
		b.Docs(mockName + " is a func-valued mock implementation of " + ifaceName + ".")
		for _, tp := range typeParams {
			b.TypeParam(tp.Name, tp.Constraint)
		}
		for _, s := range sigs {
			b.Field(s.name+"Func", funcRefFor(s), nil)
		}
		typeArgs := typeArgsFromSpecs(typeParams)
		recv := emit.Ptr(emit.Internal(b.Node(), typeArgs...))
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

// emitTypeParamSpec captures the per-parameter shape needed to
// stamp generic parameters on the emitted mock struct: the
// parameter name and its resolved constraint.
type emitTypeParamSpec struct {
	Name       string
	Constraint *emit.Constraint
}

// emitTypeParamsFromNode lifts a [node.TypeParam] slice into the
// emitTypeParamSpec slice the mock builder consumes. Constraint
// conversion runs through [refconv.ConstraintFromNode] so the
// any-constraint shape collapses to nil for round-tripping through
// the renderer's IsAny path.
func emitTypeParamsFromNode(params []*node.TypeParam) []emitTypeParamSpec {
	if len(params) == 0 {
		return nil
	}
	out := make([]emitTypeParamSpec, 0, len(params))
	for _, tp := range params {
		out = append(out, emitTypeParamSpec{
			Name:       tp.Name,
			Constraint: refconv.ConstraintFromNode(tp.Constraint),
		})
	}
	return out
}

// emitTypeParamsFromEmit projects an [emit.TypeParam] slice (the
// upstream-generator-produced interfaces this plugin consumes) into
// the spec form. The constraint passes through verbatim since it
// already lives on the emit layer.
func emitTypeParamsFromEmit(params []*emit.TypeParam) []emitTypeParamSpec {
	if len(params) == 0 {
		return nil
	}
	out := make([]emitTypeParamSpec, 0, len(params))
	for _, tp := range params {
		out = append(out, emitTypeParamSpec{Name: tp.Name, Constraint: tp.Constraint})
	}
	return out
}

// typeArgsFromSpecs lifts the local [emitTypeParamSpec] slice (the
// normalised intermediate the source-side and emit-side paths
// converge on) into the parallel bare-name [emit.Ref] list a
// generic host's receiver references take as their type arguments.
// The two layer-specific lifters live in [builder.TypeArgsFromNodeParams]
// / [builder.TypeArgsFromEmitParams]; this helper stays mockgen-local
// because [emitTypeParamSpec] is private to the package.
func typeArgsFromSpecs(specs []emitTypeParamSpec) []emit.Ref {
	if len(specs) == 0 {
		return nil
	}
	out := make([]emit.Ref, 0, len(specs))
	for _, s := range specs {
		out = append(out, emit.Builtin(s.Name))
	}
	return out
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
