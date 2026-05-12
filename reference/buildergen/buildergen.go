// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package buildergen synthesises a fluent-builder type for every
// source struct annotated with `+gen:builder`. The emitted builder
// carries one `With<Field>(value) *<Type>Builder` setter per
// exported source field plus a `Build() *<Type>` finalizer that
// returns the populated value.
//
// The plugin runs in the [priority.GeneratorFoundation] bucket so
// composition-bucket generators (e.g. mockgen) and cross-cutting
// weavers see the emitted builder type alongside repogen output.
// repogen and buildergen pointed at the same [Options.OutputPackage]
// compose into one rendered file per source struct — the two
// foundation generators share the canonical lower-cased filename
// convention so a typed-and-slot graph backed by one common Target
// resolves to a single output.
package buildergen

import (
	"errors"
	"strings"
	"unicode"

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
const Name = "buildergen"

// Capability is the capability label other plugins declare in
// [plugin.CapabilityProvider.Requires] when they want their
// generation to follow buildergen — kept in shape with repogen so
// reference plugins consistently advertise their output as a
// capability handle.
const Capability = "builder"

// DirectiveName is the bare directive name (without the `+gen:` or
// `-gen:` prefix) the plugin reads from each source struct.
const DirectiveName directive.Name = "builder"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// OutputPackage is the Go package name + directory the emitted
	// builder decls land in. Required — the plugin has no default;
	// callers configure it through
	// [pipeline.Builder.WithPluginOptions]. Pointing repogen and
	// buildergen at the same OutputPackage with shared filenames
	// composes their output into one file per source struct.
	OutputPackage string `eidos:"output_package,required"`

	// Suffix is appended to the source struct's name to form the
	// emitted builder's identifier (`<Type><Suffix>`). Defaults to
	// `Builder`.
	Suffix string `eidos:"suffix,default=Builder"`
}

// errMissingOutputPackage is returned when [Plugin.Generate] runs
// without the required OutputPackage option populated. The pipeline
// surfaces it as a positioned diagnostic on the plugin's
// configuration entry.
//
//nolint:gochecknoglobals // sentinel.
var errMissingOutputPackage = errors.New("buildergen: OutputPackage option is required")

// Plugin is the fluent-builder generator. The zero value is
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
// composition-bucket generators see emitted builder types when they
// walk the emit-store struct view.
func (*Plugin) Priority() priority.Priority { return priority.GeneratorFoundation }

// Provides returns [Capability] — composition-bucket plugins declare
// it as a Requires entry when they want documentary ordering after
// buildergen.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — buildergen has no upstream dependency.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `+gen:builder` / `-gen:builder` schema so
// directive validation rejects malformed uses at frontend-parse
// time.
func (*Plugin) Directives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(DirectiveName).
			On(node.KindStruct).
			Describe("Forces (+) or suppresses (-) builder emission for the host struct.").
			Build(),
	}
}

// Generate iterates the node store's struct bucket and emits a
// builder type for every `+gen:builder` target. Suppression via
// `-gen:builder` skips the source struct even when other generators
// act on it.
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

// shouldEmit reports whether the source struct opts into builder
// emission. Only `+gen:builder` enables; absence or `-gen:builder`
// suppresses.
func (*Plugin) shouldEmit(s *node.Struct) bool {
	return s.HasPositiveDirective(DirectiveName)
}

// emitOne appends a builder struct + its `With<Field>` setters + a
// `Build()` finalizer for one source struct. The builder carries one
// field per exported source field so accumulated state survives
// across chained calls; `Build` returns a pointer-to-source-struct
// populated from those fields.
func (p *Plugin) emitOne(pkg *builder.PackageBuilder, src *node.Struct) {
	builderName := src.Name + p.opts.Suffix
	srcRef := emit.External(src.Package, src.Name)
	target := p.targetFor(src.Name)
	exported := exportedFields(src)

	pkg.Struct(builderName, func(b *builder.StructBuilder) {
		b.Target(target)
		b.Docs(
			builderName + " accumulates field values for " + src.Name +
				" and produces the populated value via " + builderName + ".Build.",
		)
		for _, f := range exported {
			b.Field(unexport(f.Name), refconv.FromNode(f.Type), nil)
		}
		recv := func() emit.Ref { return emit.Ptr(emit.Internal(b.Node())) }
		for _, f := range exported {
			b.Method("With"+f.Name, func(m *builder.MethodBuilder) {
				m.Receiver("b", recv())
				m.Param("value", refconv.FromNode(f.Type))
				m.Return(recv())
			})
		}
		b.Method("Build", func(m *builder.MethodBuilder) {
			m.Receiver("b", recv())
			m.Return(emit.Ptr(srcRef))
		})
	})
}

// targetFor builds the canonical [emit.Target] for the supplied
// source struct: Dir + Package come from [Options.OutputPackage];
// Filename is the lower-cased source name with the standard `.go`
// extension. Matching repogen's convention lets the two foundation
// generators compose into one file when callers share OutputPackage.
func (p *Plugin) targetFor(srcName string) emit.Target {
	return emit.Target{
		Dir:      p.opts.OutputPackage,
		Filename: strings.ToLower(srcName) + ".go",
		Package:  p.opts.OutputPackage,
	}
}

// exportedFields returns the exported fields of src in declaration
// order. Unexported fields are skipped — the emitted `With<Field>`
// surface mirrors what a user-authored builder for the type could
// reach.
func exportedFields(s *node.Struct) []*node.Field {
	return s.FieldsWith(func(f *node.Field) bool {
		return f.Name != "" && unicode.IsUpper(rune(f.Name[0]))
	})
}

// unexport lower-cases the leading run of upper-case runes in name
// using Go's idiomatic initialism convention: `ID` → `id`, `URL` →
// `url`, `Name` → `name`, `URLHandler` → `urlHandler`. The result
// is the unexported builder-field identifier holding the
// accumulated value for the matching source field.
//
// The function assumes name is the identifier of an exported source
// field — non-empty and beginning with an upper-case rune — because
// the caller filters through [exportedFields] before invoking. The
// caller's invariant keeps the function's branch count minimal.
func unexport(name string) string {
	r := []rune(name)
	i := 0
	for i < len(r) && unicode.IsUpper(r[i]) {
		i++
	}
	switch {
	case i == len(r):
		return strings.ToLower(name)
	case i == 1:
		r[0] = unicode.ToLower(r[0])
		return string(r)
	default:
		return strings.ToLower(string(r[:i-1])) + string(r[i-1:])
	}
}
