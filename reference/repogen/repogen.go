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
//
// # Output layout
//
// repogen supports two layouts:
//
//   - Alongside source (default, when [Options.OutputPackage] is unset):
//     one emitted file per source struct, dropped next to the source
//     file. The router fills [emit.Target.Dir] from the source's
//     directory; the package clause matches the source package. The
//     emitted filename is `<src>_repo.go`.
//   - Centralised (when [Options.OutputPackage] is set): every emitted
//     decl lands in one Go package + directory. The filename is
//     `<src>.go` so repogen and buildergen targeting the same
//     OutputPackage compose into one file per source struct.
package repogen

import (
	"errors"
	"strings"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/core/srcfile"
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

// FilenameSuffix is appended to the source-file basename (without
// the `.go` extension) to form the alongside-source output
// filename: `<src-file>_repo.go`. The stringer-style convention
// means every `+gen:repo` struct declared in `article.go` composes
// into a single `article_repo.go`. The suffix is distinct from
// the source's own basename so re-runs don't conflate generated
// output with hand-authored code.
const FilenameSuffix = "_repo.go"

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
	// repository decls land in. When unset (the default), repogen
	// drops one `<src>_repo.go` file alongside each source struct;
	// the router resolves [emit.Target.Dir] from the source's
	// directory and the package clause matches the source. Set
	// OutputPackage when the project keeps generated code in a
	// dedicated sibling package.
	OutputPackage string `eidos:"output_package"`

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
//
// Output layout depends on [Options.OutputPackage]:
//
//   - Unset (default): one emit.Package per source package, emitted
//     alongside the source files.
//   - Set: every decl lands in a single centralised emit.Package.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	if p.opts.OutputPackage != "" {
		return p.generateCentralised(ctx)
	}
	return p.generateAlongsideSource(ctx)
}

// generateCentralised drops every emitted decl into one shared
// emit.Package keyed by [Options.OutputPackage]. The behaviour
// downstream consumers depended on before alongside-source routing
// existed.
func (p *Plugin) generateCentralised(ctx *plugin.GeneratorContext) error {
	c := builder.For(Name, emit.Target{})
	pkg := c.Package(p.opts.OutputPackage, Name+":"+p.opts.OutputPackage)
	ctx.Reader.Structs().Each(func(s *node.Struct) {
		if !p.shouldEmit(s) {
			return
		}
		target := emit.Target{
			Dir:      p.opts.OutputPackage,
			Filename: strings.ToLower(s.Name) + ".go",
			Package:  p.opts.OutputPackage,
		}
		p.emitOne(pkg, s, target, emit.External(s.Package, s.Name))
	})
	out, err := pkg.Build()
	if err != nil {
		return err
	}
	if err := ctx.Store.Emit().AddPackage(out); err != nil {
		return errors.Join(errAddPackage, err)
	}
	return nil
}

// generateAlongsideSource emits one emit.Package per source package,
// containing one decl per `+gen:repo` target. Each emitted decl's
// Target is filename-only — the router fills Target.Dir from the
// source's directory at the routing phase — and carries the source
// package's import path so the renderer elides same-package
// qualifiers via the [emit.Target.ImportPath] mechanism. The plugin
// therefore emits every type reference (including the source
// struct's own type) as [emit.External]; the backend renders
// cross-package refs qualified and same-package refs bare.
func (p *Plugin) generateAlongsideSource(ctx *plugin.GeneratorContext) error {
	var firstErr error
	ctx.Reader.Packages().Each(func(srcPkg *node.Package) {
		matches := matchingStructs(srcPkg, p.shouldEmit)
		if len(matches) == 0 {
			return
		}
		c := builder.For(Name, emit.Target{})
		pkg := c.Package(srcPkg.Name, Name+":"+srcPkg.Path)
		for _, s := range matches {
			target := emit.Target{
				Filename:   srcfile.WithSuffix(s.Pos(), s.Name, FilenameSuffix),
				Package:    srcPkg.Name,
				ImportPath: srcPkg.Path,
			}
			p.emitOne(pkg, s, target, emit.External(s.Package, s.Name))
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

// errAddPackage is the sentinel wrapped around store-side AddPackage
// failures. Tests and callers detect the class with errors.Is.
//
//nolint:gochecknoglobals // sentinel.
var errAddPackage = errors.New("repogen: add package to store")

// matchingStructs returns srcPkg's structs the predicate accepts, in
// declaration order. Extracted so both layouts share the predicate
// path and the per-package slice allocation pattern.
func matchingStructs(srcPkg *node.Package, ok func(*node.Struct) bool) []*node.Struct {
	out := make([]*node.Struct, 0, len(srcPkg.Structs))
	for _, s := range srcPkg.Structs {
		if ok(s) {
			out = append(out, s)
		}
	}
	return out
}

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
//
// srcRef is the reference the emitted methods use for the source
// type — callers in centralised layout pass [emit.External] so the
// renderer adds the import; callers in alongside-source layout pass
// [emit.Builtin] so the renderer emits a bare identifier without a
// self-import.
//
// The Origin of each emitted decl is set to src so the router can
// resolve Target.Dir from the source's file path when target.Dir
// is empty (alongside-source layout).
func (p *Plugin) emitOne(pkg *builder.PackageBuilder, src *node.Struct, target emit.Target, srcRef emit.Ref) {
	ifaceName := p.identifier(src.Name + p.opts.InterfaceSuffix)
	structName := p.identifier(src.Name + p.opts.StructSuffix)

	pkg.Interface(ifaceName, func(i *builder.InterfaceBuilder) {
		i.Target(target)
		i.Origin(src)
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
		st.Origin(src)
		st.Docs(structName + " is the default in-memory implementation of " + ifaceName + ".")
		st.Method(p.identifier("Get"), func(m *builder.MethodBuilder) {
			m.Receiver("r", emit.Ptr(emit.Internal(st.Node())))
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("id", emit.Builtin("string"))
			m.Return(emit.Ptr(srcRef))
			m.Return(emit.Builtin("error"))
			m.Body(emit.NewReturn(emit.NewLiteralNil(), emit.NewLiteralNil()))
		})
		st.Method(p.identifier("List"), func(m *builder.MethodBuilder) {
			m.Receiver("r", emit.Ptr(emit.Internal(st.Node())))
			m.Param("ctx", emit.External("context", "Context"))
			m.Return(emit.SliceOf(emit.Ptr(srcRef)))
			m.Return(emit.Builtin("error"))
			m.Body(emit.NewReturn(emit.NewLiteralNil(), emit.NewLiteralNil()))
		})
		st.Method(p.identifier("Save"), func(m *builder.MethodBuilder) {
			m.Receiver("r", emit.Ptr(emit.Internal(st.Node())))
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("value", emit.Ptr(srcRef))
			m.Return(emit.Builtin("error"))
			m.Body(emit.NewReturn(emit.NewLiteralNil()))
		})
		st.Method(p.identifier("Delete"), func(m *builder.MethodBuilder) {
			m.Receiver("r", emit.Ptr(emit.Internal(st.Node())))
			m.Param("ctx", emit.External("context", "Context"))
			m.Param("id", emit.Builtin("string"))
			m.Return(emit.Builtin("error"))
			m.Body(emit.NewReturn(emit.NewLiteralNil()))
		})
	})
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
