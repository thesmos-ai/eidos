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
package registrygen

import (
	"embed"
	"errors"
	"io/fs"
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

// Options carries the plugin's user-tunable settings.
type Options struct {
	// OutputPackage is the Go package name + directory the
	// emitted registry init block lands in. Required.
	OutputPackage string `eidos:"output_package,required"`

	// Filename is the rendered file name registrations are
	// collected into. Defaults to `registry.go`.
	Filename string `eidos:"filename,default=registry.go"`
}

// errMissingOutputPackage is returned when [Plugin.Generate] runs
// without the required OutputPackage option populated.
//
//nolint:gochecknoglobals // sentinel.
var errMissingOutputPackage = errors.New("registry-gen: OutputPackage option is required")

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
// renders it as a single-line `registry.Register("<Name>", <Init>)`
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
	// the value argument to `registry.Register`. Generators
	// produce composite literals (`Article{}`), constructor calls,
	// or any other expression the registry accepts.
	Init *emit.Expr

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
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	if p.opts.OutputPackage == "" {
		return errMissingOutputPackage
	}
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
		reg := &Registration{
			Name:    s.Name,
			NameLit: emit.NewLiteralString(s.Name),
			Init: emit.NewComposite(
				emit.External(s.Package, s.Name),
				nil,
			),
			Target: target,
		}
		// Init slot has no element-kind constraint, so the
		// plugin-defined kind reaches the renderer where the
		// shipped template dispatches it. Append is infallible
		// for non-nil items on a kindless slot.
		_ = file.Init().Append(reg, c.Provenance("registry."+s.Name))
	}
	return nil
}
