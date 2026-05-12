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
// # Output routing
//
// mockgen owns no routing configuration of its own — the
// framework's routing layer composes every emit decl's
// [emit.Target] from the source struct's origin plus the project
// / per-plugin output config and CLI overrides. The plugin
// declares its filename suffix via [Plugin.FilenameSuffix] and
// sets [emit.BaseEmit.OriginNode] on every emit decl; the
// Layout phase does the rest.
//
// [Options.Test] is plugin behaviour, not a routing decision:
// the Test toggle flips the declared filename suffix to
// [TestFilenameSuffix] so the rendered mock lands in a `_test.go`
// file. Users who want mocks in a sibling `<srcPkg>_test`
// package set the per-plugin `output.package` config alongside
// `options.test: true` — two independent toggles for two
// independent concerns.
package mockgen

import (
	"errors"
	"strconv"

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

// FilenameSuffix is appended to the source-file basename (without
// the `.go` extension) to form the alongside-source output
// filename: `<src-file>_mock.go`. Every `+gen:mock` interface
// declared in `searcher.go` therefore composes into one
// `searcher_mock.go`.
const FilenameSuffix = "_mock.go"

// TestFilenameSuffix is the [Plugin.FilenameSuffix] value when
// [Options.Test] is set: `<src-file>_test.go`. Go's test toolchain
// compiles `_test.go` files only at test time, so the generated
// mock stays out of production builds.
const TestFilenameSuffix = "_test.go"

// Language is the backend language whose suffixes
// [Plugin.FilenameSuffix] returns. Other languages get the empty
// signal until matching templates and per-language suffix
// configuration land.
const Language = "golang"

// Options carries the plugin's user-tunable settings. Routing is
// owned by the framework's routing layer; mockgen exposes no
// output-package option — project / per-plugin output config and
// the CLI `-layout` / `-p` / `-output-dir` flags drive where
// rendered output lands.
type Options struct {
	// Suffix is appended to the targeted interface's name to form
	// the emitted mock struct's identifier (`<Type><Suffix>`).
	// Defaults to `Mock`.
	Suffix string `eidos:"suffix,default=Mock"`

	// Test selects the test-file filename suffix
	// ([TestFilenameSuffix]) so the rendered mock lands in a
	// `_test.go` file and the Go test toolchain confines it to
	// test builds. Test only affects the declared filename suffix
	// — the rendered package and import path remain whatever the
	// framework's routing layer composes from the project /
	// per-plugin `output.package` config and CLI overrides.
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

// FilenameSuffix returns the per-source filename suffix the Layout
// phase appends to the source's basename when composing the
// rendered output path for every decl this plugin emits.
// Implements [plugin.FilenameProvider]. The plugin emits standard
// Go decls today; consumers targeting another backend language
// receive the empty signal so the Layout phase surfaces a
// missing-FilenameProvider error rather than producing a Go
// suffix that wouldn't match the rendered output.
//
// [Options.Test] switches the suffix to [TestFilenameSuffix] so
// the rendered mock lands in a `_test.go` file. The switch is
// per-mode behaviour: every emission from a Test-enabled plugin
// instance carries the test suffix.
func (p *Plugin) FilenameSuffix(lang string) string {
	if lang != Language {
		return ""
	}
	if p.opts.Test {
		return TestFilenameSuffix
	}
	return FilenameSuffix
}

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
// plugin walks two sources: source-side interfaces carrying
// `+gen:mock` (one mock per match, anchored to the source
// interface) and emit-store interfaces (one mock per upstream
// interface that hasn't opted out via `-gen:mock`, anchored to
// the upstream interface's Origin so the mock composes into the
// same file as the interface it shadows).
//
// Routing is owned entirely by the framework's routing layer:
// every emit decl carries Origin set and leaves [emit.Target]
// zero. The Layout phase composes Dir / Filename / Package /
// ImportPath from the origin and the resolved
// [pipeline.LayoutPolicy] for this plugin.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	var firstErr error
	ctx.Reader.Packages().Each(func(srcPkg *node.Package) {
		matches := matchingSourceInterfaces(srcPkg)
		if len(matches) == 0 {
			return
		}
		c := builder.For(Name, emit.Target{})
		pkg := c.Package(srcPkg.Name, srcPkg.Path)
		for _, si := range matches {
			p.emitForSourceInterface(pkg, si, emit.External(si.Package, si.Name))
		}
		if err := buildAndAdd(ctx, pkg); err != nil && firstErr == nil {
			firstErr = err
		}
	})
	ctx.Reader.EmitPackages().Each(func(epkg *emit.Package) {
		matches := matchingEmitInterfaces(epkg)
		if len(matches) == 0 {
			return
		}
		c := builder.For(Name, emit.Target{})
		pkg := c.Package(epkg.Name, epkg.Path)
		for _, ei := range matches {
			p.emitForEmitInterface(pkg, ei)
		}
		if err := buildAndAdd(ctx, pkg); err != nil && firstErr == nil {
			firstErr = err
		}
	})
	return firstErr
}

// buildAndAdd finalises the in-progress package and folds it into
// the emit store. Wrap-and-Join keeps the call sites in [Generate]
// uniform across the source-side and emit-side passes.
func buildAndAdd(ctx *plugin.GeneratorContext, pkg *builder.PackageBuilder) error {
	out, err := pkg.Build()
	if err != nil {
		return err
	}
	if err := ctx.Store.Emit().AddPackage(out); err != nil {
		return errors.Join(errAddPackage, err)
	}
	return nil
}

// errAddPackage is the sentinel wrapped around store-side AddPackage
// failures. Tests and callers detect the class with errors.Is.
//
//nolint:gochecknoglobals // sentinel.
var errAddPackage = errors.New("mockgen: add package to store")

// matchingSourceInterfaces returns srcPkg's source-side interfaces
// opted into mocking by `+gen:mock`.
func matchingSourceInterfaces(srcPkg *node.Package) []*node.Interface {
	out := make([]*node.Interface, 0, len(srcPkg.Interfaces))
	for _, i := range srcPkg.Interfaces {
		if i.HasPositiveDirective(DirectiveName) {
			out = append(out, i)
		}
	}
	return out
}

// matchingEmitInterfaces returns epkg's emit-side interfaces that
// haven't opted out via `-gen:mock`. The default-on polarity
// mirrors the documented contract: upstream generators land an
// interface and mockgen mocks it unless the user explicitly
// suppresses.
func matchingEmitInterfaces(epkg *emit.Package) []*emit.Interface {
	out := make([]*emit.Interface, 0, len(epkg.Interfaces))
	for _, i := range epkg.Interfaces {
		if i.HasNegatedDirective(DirectiveName) {
			continue
		}
		out = append(out, i)
	}
	return out
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
// interface anchored to the upstream interface's Origin. The Mock
// references the source interface by [emit.Internal] so the
// rendered struct correctly resolves the in-target name regardless
// of the emit interface's own package. Generic emit interfaces
// propagate their type parameters to the mock so the rendered
// struct, methods, and ifaceRef-instantiation all thread
// `[T1, T2, …]` consistently.
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
	tps := emitTypeParamsFromEmit(i.TypeParams)
	typeArgs := builder.TypeArgsFromEmitParams(i.TypeParams)
	p.emitMock(pkg, i.Name, emit.Internal(i, typeArgs...), i.Origin(), tps, sigs)
}

// emitForSourceInterface emits a Mock struct for a source-side
// interface, lifting its node-layer types into emit refs through
// [refconv.FromNode] so the generated mock parses against the same
// signatures the source declares. ifaceRef is the reference the
// emitted mock uses for the source interface — the plugin always
// passes [emit.External]; the renderer elides same-package
// qualifiers via the Target.ImportPath the Layout phase composes
// from the source package.
//
// Generic source interfaces propagate their type parameters to the
// mock and thread the type-arg list through ifaceRef so the
// rendered struct, methods, and embedded reference all carry
// `[T1, T2, …]` consistently.
func (p *Plugin) emitForSourceInterface(
	pkg *builder.PackageBuilder,
	i *node.Interface,
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
	p.emitMock(pkg, i.Name, builder.ApplyTypeArgs(ifaceRef, typeArgs), i, tps, sigs)
}

// emitMock appends one Mock struct decl carrying the func-valued
// fields and dispatching methods for every signature in sigs. The
// mocked interface is rendered through ifaceRef in field-receiver
// position so callers can pass the Mock anywhere the source
// interface is required. typeParams (when non-empty) propagate the
// host interface's generic parameters to the mock so the rendered
// struct, methods, and receivers all carry the same `[T1, T2, …]`
// bracket list.
//
// Origin is set on the emitted struct so the Layout phase can
// resolve every Target field downstream — the plugin itself
// never constructs an [emit.Target] literal.
func (p *Plugin) emitMock(
	pkg *builder.PackageBuilder,
	ifaceName string,
	ifaceRef emit.Ref,
	origin node.Node,
	typeParams []emitTypeParamSpec,
	sigs []methodSig,
) {
	_ = ifaceRef // reserved for future "var _ Iface = (*Mock)(nil)" emission.
	mockName := ifaceName + p.opts.Suffix

	pkg.Struct(mockName, func(b *builder.StructBuilder) {
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
