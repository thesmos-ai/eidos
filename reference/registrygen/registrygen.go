// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package registrygen showcases the plugin-defined-emit-kind path:
// it defines its own [Registration] emit kind outside the
// `emit.*` namespace, ships the matching `registration.tmpl`
// template through [plugin.TemplateProvider], and appends one
// Registration to a target file's [emit.File.Init] slot for each
// source struct annotated with `+gen:register`. The rendered
// output collects every registration into a single `func init()`
// block per file.
//
// # Output layout
//
// registrygen supports two layouts:
//
//   - Alongside source (default, when [Options.OutputPackage] is
//     unset): one `registry.go` per source package, dropped next to
//     the package's source files. Each per-package file collects the
//     registrations declared in that package.
//   - Centralised (when [Options.OutputPackage] is set): every
//     registration lands in one `<OutputPackage>/<Filename>` file,
//     useful when downstream code expects to import one registry.
package registrygen

import (
	"embed"
	"errors"
	"io/fs"
	"path/filepath"
	"text/template"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
)

// Name is the plugin's stable identifier.
const Name = "registry-gen"

// Capability is the capability label this plugin advertises.
const Capability = "registry"

// DirectiveName is the bare directive name read from source
// structs.
const DirectiveName directive.Name = "register"

// Kind is the plugin-defined emit kind every [Registration] reports
// from [Registration.Kind]. The dotted spelling keeps it outside
// the core `emit.*` namespace.
const Kind directive.Kind = "registrygen.registration"

// Language is the target language whose template tree the plugin
// contributes. Other languages get the empty signal from
// [Plugin.Templates].
const Language = "golang"

// DefaultFilename is the rendered output's filename when
// [Options.Filename] is unset. The same name is used in both layouts
// so the registration block is always recognisable.
const DefaultFilename = "registry.go"

// DefaultRegisterPackage is the import path the rendered
// register-call resolves to when [Options.RegisterPackage] is
// unset. Stdlib `log` keeps the demo self-contained so a fresh
// project produces compilable output without external setup;
// production deployments override RegisterPackage + RegisterFunc
// to point at the real registry surface (typically
// `example.com/registry` exposing `Register(name string, value
// any)` or similar).
const DefaultRegisterPackage = "log"

// DefaultRegisterFunc is the function name called on the registry
// package when [Options.RegisterFunc] is unset. `log.Print`
// accepts a variadic `...any` argument list so the
// `Func(name, value)` shape stays valid both for stdlib log and
// for a real `Register(name, value)` function — no shape
// translation needed when callers switch packages.
const DefaultRegisterFunc = "Print"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// OutputPackage is the Go package name + directory the emitted
	// registry init block lands in. When unset (the default), the
	// plugin drops one registry file per source package alongside
	// the package's source files; set OutputPackage to centralise
	// every registration into a single sibling package.
	OutputPackage string `eidos:"output_package"`

	// Filename is the rendered file name registrations are
	// collected into. Defaults to [DefaultFilename].
	Filename string `eidos:"filename,default=registry.go"`

	// RegisterPackage is the import path of the registry package
	// the rendered call references. Defaults to
	// [DefaultRegisterPackage]. The renderer registers the import
	// on the host file's import set via the [emit.NewExternal]
	// expression — no plugin-side import scaffolding needed.
	RegisterPackage string `eidos:"register_package,default=log"`

	// RegisterFunc is the function name called on the registry
	// package. The rendered call passes two positional arguments —
	// the struct's name (string) and a composite literal of the
	// struct — so the configured Func must accept that shape (or
	// a variadic `...any` like stdlib `log.Print`). Defaults to
	// [DefaultRegisterFunc].
	RegisterFunc string `eidos:"register_func,default=Print"`
}

// Plugin is the registry-gen generator. Go through [New] so the
// embedded [opt.Holder] binds to the plugin's options field.
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

// Priority places the plugin in the cross-cutting bucket so it
// runs after foundation and composition generators.
func (*Plugin) Priority() priority.Priority { return priority.GeneratorCrossCutting }

// Provides advertises the registry capability.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — registry-gen has no upstream dependency.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `+gen:register` schema.
func (*Plugin) Directives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(DirectiveName).
			On(node.KindStruct).
			Describe("Registers the host struct with the runtime registry on package init.").
			Build(),
	}
}

//go:embed templates/golang/*.tmpl
var templatesFS embed.FS

// Templates returns the embedded template filesystem for the
// requested language. Only `golang` is shipped today; other
// languages receive the empty signal. The fs.Sub call cannot fail
// because the subdirectory is guaranteed by the build-time
// `go:embed` directive.
func (*Plugin) Templates(lang string) (fs.FS, bool) {
	if lang != Language {
		return nil, false
	}
	sub, _ := fs.Sub(templatesFS, "templates/"+lang)
	return sub, true
}

// TemplateFuncs returns nil — the plugin ships no funcmap
// extensions.
func (*Plugin) TemplateFuncs(string) template.FuncMap { return nil }

// TemplateOverrides returns nil — the plugin ships no funcmap
// overrides.
func (*Plugin) TemplateOverrides(string) template.FuncMap { return nil }

// Registration is the plugin-defined emit kind every emitted
// registration carries. The matching `registration.tmpl` template
// renders it as a single-line `<RegisterFunc>(<NameLit>, <Init>)`
// call — slotted into a file's [emit.File.Init] block so all
// registrations land inside one `func init() { ... }` per file.
type Registration struct {
	emit.BaseEmit

	// Name is the source struct's identifier — also the key the
	// registry records the value under.
	Name string

	// NameLit is a pre-built [emit.Expr] string literal carrying
	// the name in quoted form. Exposed so the template can render
	// it via `renderExpr` without needing to know how to escape.
	NameLit *emit.Expr

	// Init is the expression evaluated at init time and passed as
	// the value argument to the register call. Generators produce
	// composite literals (`Article{}`), constructor calls, or any
	// other expression the registry accepts.
	Init *emit.Expr

	// RegisterFunc is the register-call's callee — built via
	// [emit.NewExternal] so the renderer registers the configured
	// import path with the rendered file's import set without
	// any plugin-side import scaffolding. Defaults to a reference
	// into stdlib `log.Print`; configurable through
	// [Options.RegisterPackage] / [Options.RegisterFunc].
	RegisterFunc *emit.Expr

	// Target identifies the file the registration contributes to.
	Target emit.Target
}

// Kind returns [Kind].
func (*Registration) Kind() directive.Kind { return Kind }

// Compile-time confirmation that *Registration is a valid
// [emit.Node].
var _ emit.Node = (*Registration)(nil)

// Generate iterates the node store's struct bucket and appends one
// Registration to the target file's Init slot for each
// `+gen:register` annotated struct.
//
// Output layout depends on [Options.OutputPackage]:
//
//   - Unset (default): one `registry.go` per source package,
//     emitted alongside that package's sources.
//   - Set: every Registration lands in a single centralised file.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	if p.opts.OutputPackage != "" {
		return p.generateCentralised(ctx)
	}
	return p.generateAlongsideSource(ctx)
}

// generateCentralised collects every `+gen:register` struct into
// one shared file keyed by [Options.OutputPackage]. The registered
// value uses an [emit.External] composite literal so the rendered
// file imports the source's package and references it qualified.
func (p *Plugin) generateCentralised(ctx *plugin.GeneratorContext) error {
	target := emit.Target{
		Dir:      p.opts.OutputPackage,
		Filename: p.opts.Filename,
		Package:  p.opts.OutputPackage,
	}
	c := builder.For(Name, target)
	// FileFor only errors when the EmitView is frozen; Generate
	// runs pre-freeze so the call is infallible here.
	file, _ := ctx.Store.Emit().FileFor(target)
	for _, s := range ctx.Reader.Structs().Slice() {
		if !s.HasPositiveDirective(DirectiveName) {
			continue
		}
		p.appendRegistration(file, c, s, target, emit.External(s.Package, s.Name))
	}
	return nil
}

// generateAlongsideSource emits one registry file per source
// package, dropped alongside the package's source files. The
// Target.Dir is derived from the first source file's directory so
// the rendered file shares its location with the package it
// registers.
func (p *Plugin) generateAlongsideSource(ctx *plugin.GeneratorContext) error {
	var firstErr error
	ctx.Reader.Packages().Each(func(srcPkg *node.Package) {
		matches := matchingStructs(srcPkg)
		if len(matches) == 0 {
			return
		}
		dir, ok := packageDir(srcPkg)
		if !ok {
			if firstErr == nil {
				firstErr = errors.Join(
					errMissingPackageDir,
					errors.New("source package "+srcPkg.Path+" has no source-file attribution"),
				)
			}
			return
		}
		target := emit.Target{
			Dir:      dir,
			Filename: p.opts.Filename,
			Package:  srcPkg.Name,
		}
		c := builder.For(Name, target)
		file, _ := ctx.Store.Emit().FileFor(target)
		for _, s := range matches {
			// Source lives in the same Go package as the rendered
			// init block — use a bare identifier so the renderer
			// doesn't insert a self-import.
			p.appendRegistration(file, c, s, target, emit.Builtin(s.Name))
		}
	})
	return firstErr
}

// matchingStructs returns srcPkg's structs opted into registration
// by `+gen:register`.
func matchingStructs(srcPkg *node.Package) []*node.Struct {
	out := make([]*node.Struct, 0, len(srcPkg.Structs))
	for _, s := range srcPkg.Structs {
		if s.HasPositiveDirective(DirectiveName) {
			out = append(out, s)
		}
	}
	return out
}

// packageDir returns the directory containing srcPkg's first source
// file, used as the alongside-source layout's Target.Dir. Returns
// the empty string and false when srcPkg has no source-file
// attribution — synthetic packages without files can't be routed
// alongside source.
func packageDir(srcPkg *node.Package) (string, bool) {
	for _, f := range srcPkg.Files {
		if f.Path != "" {
			return filepath.Dir(f.Path), true
		}
	}
	return "", false
}

// appendRegistration creates one Registration for src and appends
// it to file's Init slot, attributing the contribution through c's
// provenance helper. typeRef is the reference the registered value's
// composite literal uses for the source type — callers in
// centralised layout pass [emit.External]; callers in
// alongside-source layout pass [emit.Builtin] so the renderer
// doesn't insert a self-import.
//
// The Registration's [Registration.RegisterFunc] resolves to
// `<Options.RegisterPackage>.<Options.RegisterFunc>` via
// [emit.NewExternal]; the renderer registers the configured
// package's import on the rendered file's import set automatically.
func (p *Plugin) appendRegistration(
	file *emit.File,
	c *builder.Context,
	src *node.Struct,
	target emit.Target,
	typeRef emit.Ref,
) {
	reg := &Registration{
		Name:         src.Name,
		NameLit:      emit.NewLiteralString(src.Name),
		Init:         emit.NewComposite(typeRef, nil),
		RegisterFunc: emit.NewExternal(p.registerPackage(), p.registerFunc()),
		Target:       target,
	}
	// Init slot has no element-kind constraint, so the plugin-
	// defined kind reaches the renderer where the shipped template
	// dispatches it. Append is infallible for non-nil items on a
	// kindless slot.
	_ = file.Init().Append(reg, c.Provenance("registry."+src.Name))
}

// registerPackage / registerFunc return the configured option value
// or the documented default when the option is empty. Centralised
// so the rendered behaviour stays consistent across both layouts
// even when a caller bypasses SetOptions.
func (p *Plugin) registerPackage() string {
	if p.opts.RegisterPackage != "" {
		return p.opts.RegisterPackage
	}
	return DefaultRegisterPackage
}

func (p *Plugin) registerFunc() string {
	if p.opts.RegisterFunc != "" {
		return p.opts.RegisterFunc
	}
	return DefaultRegisterFunc
}

// errMissingPackageDir surfaces when alongside-source routing is
// active and a source package carries no resolvable directory —
// typically a synthetic package without source files.
//
//nolint:gochecknoglobals // sentinel.
var errMissingPackageDir = errors.New("registry-gen: source package has no resolvable directory")
