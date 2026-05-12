// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package repogen synthesises a Repository-pattern interface and a
// thin implementing struct for every source struct annotated with
// `+gen:repo`. The emitted interface carries the canonical CRUD
// method set (`Get` / `List` / `Save` / `Delete`); the implementing
// struct carries empty bodies — downstream cross-cutting plugins
// fill the bodies via the method `prebody` / `postbody` slots.
//
// Detection is directive-driven; the heuristic-driven shape-writer
// pattern doesn't apply. `+gen:repo` opts a struct into emission;
// `-gen:repo` suppresses it.
package repogen

import (
	"errors"
	"strings"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
)

// Name is the plugin's stable identifier surfaced through
// [plugin.Plugin.Name].
const Name = "repogen"

// Capability is the capability label other plugins declare in
// [plugin.CapabilityProvider.Requires] when they need repogen's
// output as input — most notably the mock generator.
const Capability = "repository"

// DirectiveName is the bare directive name (without the `+gen:` or
// `-gen:` prefix) the plugin reads from each source struct.
const DirectiveName directive.Name = "repo"

// NamingPascal selects the canonical PascalCase identifier shape
// (e.g. `ArticleRepository`, `Get`). Default for the Naming option.
const NamingPascal = "Pascal"

// NamingCamel selects camelCase identifier shape (e.g.
// `articleRepository`, `get`) — produces unexported emitted
// identifiers, useful for internal repository surfaces.
const NamingCamel = "Camel"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// OutputPackage is the Go package name + directory the emitted
	// repository decls land in. Required — the plugin has no
	// default; callers configure it through
	// [pipeline.Builder.WithPluginOptions].
	OutputPackage string `eidos:"output_package,required"`

	// InterfaceSuffix is appended to the source struct's name to
	// form the emitted interface's identifier
	// (`<Type><InterfaceSuffix>`). Defaults to `Repository`.
	InterfaceSuffix string `eidos:"interface_suffix,default=Repository"`

	// StructSuffix is appended to the source struct's name to form
	// the emitted implementing-struct's identifier
	// (`<Type><StructSuffix>`). Defaults to `Repo`.
	StructSuffix string `eidos:"struct_suffix,default=Repo"`

	// Naming selects the casing style for emitted identifiers.
	// `Pascal` produces exported identifiers; `Camel` produces
	// unexported (lowercase-first) identifiers. Defaults to
	// `Pascal`.
	Naming string `eidos:"naming,one_of=Pascal|Camel,default=Pascal"`
}

// Plugin is the repository-pattern generator. The zero value is
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

// Priority places the plugin in the foundation generator bucket so
// it runs before composition-bucket generators (e.g. mockgen) that
// consume its emitted interfaces.
func (*Plugin) Priority() priority.Priority { return priority.GeneratorFoundation }

// Provides returns [Capability] — composition-bucket plugins
// declare it as a Requires entry to document the dependency.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — repogen has no upstream dependency.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `+gen:repo` / `-gen:repo` schema with
// the pipeline so directive validation rejects malformed uses at
// frontend-parse time.
func (*Plugin) Directives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(DirectiveName).
			On(node.KindStruct).
			Describe("Forces (+) or suppresses (-) repository emission for the host struct.").
			Build(),
	}
}

// Generate iterates the node store's struct bucket, emitting a
// Repository interface + implementing struct for each `+gen:repo`
// target. Suppression via `-gen:repo` skips the source struct
// even when other generators (builder, registry) act on it.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	if p.opts.OutputPackage == "" {
		return errMissingOutputPackage
	}
	c := builder.For(Name, emit.Target{})
	// Path is plugin-namespaced so multiple foundation generators
	// sharing one OutputPackage do not collide on the emit store's
	// unique-path key. Routing to the rendered file still goes
	// through each decl's Target.
	pkg := c.Package(p.opts.OutputPackage, Name+":"+p.opts.OutputPackage)
	ctx.Reader.Structs().Each(func(s *node.Struct) {
		if !p.shouldEmit(s) {
			return
		}
		p.emitOne(pkg, s)
	})
	out, err := pkg.Build()
	if err != nil {
		return err
	}
	return ctx.Store.Emit().AddPackage(out)
}

// errMissingOutputPackage is returned when [Plugin.Generate] runs
// without the required OutputPackage option populated. The
// pipeline surfaces it as a positioned diagnostic on the plugin's
// configuration entry.
var errMissingOutputPackage = errors.New("repogen: OutputPackage option is required")

// shouldEmit reports whether the source struct opts into
// repository emission. Only `+gen:repo` enables; absence or
// `-gen:repo` suppresses.
func (*Plugin) shouldEmit(s *node.Struct) bool {
	return s.HasPositiveDirective(DirectiveName)
}

// emitOne appends the interface + implementing-struct + method
// set for one source struct to the in-progress package. The
// emitted method signatures cover the canonical CRUD set with
// `context.Context` first parameters and `error` returns; bodies
// are empty so cross-cutting weavers have a clean slot surface.
func (p *Plugin) emitOne(pkg *builder.PackageBuilder, src *node.Struct) {
	ifaceName := p.identifier(src.Name + p.opts.InterfaceSuffix)
	structName := p.identifier(src.Name + p.opts.StructSuffix)
	srcRef := emit.External(src.Package, src.Name)
	target := p.targetFor(src.Name)

	pkg.Interface(ifaceName, func(i *builder.InterfaceBuilder) {
		i.Target(target)
		i.Docs(ifaceName + " stores and retrieves " + src.Name + " values.")
		i.Method(p.identifier("Get"), func(m *builder.MethodBuilder) {
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("id", emit.Builtin("string"))
			m.Return(emit.Ptr(srcRef))
			m.Return(emit.Builtin("error"))
		})
		i.Method(p.identifier("List"), func(m *builder.MethodBuilder) {
			m.Param("ctx", emit.External("context", "Context"))
			m.Return(emit.SliceOf(emit.Ptr(srcRef)))
			m.Return(emit.Builtin("error"))
		})
		i.Method(p.identifier("Save"), func(m *builder.MethodBuilder) {
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("value", emit.Ptr(srcRef))
			m.Return(emit.Builtin("error"))
		})
		i.Method(p.identifier("Delete"), func(m *builder.MethodBuilder) {
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("id", emit.Builtin("string"))
			m.Return(emit.Builtin("error"))
		})
	})

	pkg.Struct(structName, func(st *builder.StructBuilder) {
		st.Target(target)
		st.Docs(structName + " is the default in-memory implementation of " + ifaceName + ".")
		st.Method(p.identifier("Get"), func(m *builder.MethodBuilder) {
			m.Receiver("r", emit.Ptr(emit.Internal(st.Node())))
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("id", emit.Builtin("string"))
			m.Return(emit.Ptr(srcRef))
			m.Return(emit.Builtin("error"))
		})
		st.Method(p.identifier("List"), func(m *builder.MethodBuilder) {
			m.Receiver("r", emit.Ptr(emit.Internal(st.Node())))
			m.Param("ctx", emit.External("context", "Context"))
			m.Return(emit.SliceOf(emit.Ptr(srcRef)))
			m.Return(emit.Builtin("error"))
		})
		st.Method(p.identifier("Save"), func(m *builder.MethodBuilder) {
			m.Receiver("r", emit.Ptr(emit.Internal(st.Node())))
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("value", emit.Ptr(srcRef))
			m.Return(emit.Builtin("error"))
		})
		st.Method(p.identifier("Delete"), func(m *builder.MethodBuilder) {
			m.Receiver("r", emit.Ptr(emit.Internal(st.Node())))
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("id", emit.Builtin("string"))
			m.Return(emit.Builtin("error"))
		})
	})
}

// targetFor builds the canonical [emit.Target] for the supplied
// source struct: Dir + Package come from [Options.OutputPackage];
// Filename is the lower-cased source name with the standard `.go`
// extension. The shared convention keeps repogen and buildergen
// output for the same source struct composing into one file when
// callers pick the same OutputPackage.
func (p *Plugin) targetFor(srcName string) emit.Target {
	return emit.Target{
		Dir:      p.opts.OutputPackage,
		Filename: strings.ToLower(srcName) + ".go",
		Package:  p.opts.OutputPackage,
	}
}

// identifier applies the configured naming style. Pascal returns
// the name unchanged; Camel lower-cases the first rune so the
// emitted identifier is unexported.
func (p *Plugin) identifier(name string) string {
	if p.opts.Naming != NamingCamel || name == "" {
		return name
	}
	return string(name[0]|0x20) + name[1:]
}
