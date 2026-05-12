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
//
// # Output layout
//
// buildergen supports two layouts:
//
//   - Alongside source (default, when [Options.OutputPackage] is unset):
//     one `<src>_builder.go` per source struct, dropped next to the
//     source file. The router fills [emit.Target.Dir] from the
//     source's directory; the package clause matches the source
//     package.
//   - Centralised (when [Options.OutputPackage] is set): every
//     emitted decl lands in one Go package + directory. The filename
//     is `<src>.go` so repogen and buildergen targeting the same
//     OutputPackage compose into one file per source struct.
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
// generation to follow buildergen.
const Capability = "builder"

// DirectiveName is the bare directive name (without the `+gen:` or
// `-gen:` prefix) the plugin reads from each source struct.
const DirectiveName directive.Name = "builder"

// FilenameSuffix is appended to the lower-cased source struct name
// to form the alongside-source output filename: `<src>_builder.go`.
// Distinct from the source file so the second run doesn't conflate
// generated output with hand-written code.
const FilenameSuffix = "_builder.go"

// Options carries the plugin's user-tunable settings.
type Options struct {
	// OutputPackage is the Go package name + directory the emitted
	// builder decls land in. When unset (the default), buildergen
	// drops one `<src>_builder.go` file alongside each source struct;
	// the router resolves [emit.Target.Dir] from the source's
	// directory and the package clause matches the source. Set
	// OutputPackage when the project keeps generated code in a
	// dedicated sibling package — pointing repogen and buildergen at
	// the same OutputPackage composes their output into one file per
	// source struct.
	OutputPackage string `eidos:"output_package"`

	// Suffix is appended to the source struct's name to form the
	// emitted builder's identifier (`<Type><Suffix>`). Defaults to
	// `Builder`.
	Suffix string `eidos:"suffix,default=Builder"`
}

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
// emit.Package keyed by [Options.OutputPackage].
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
// containing one decl per `+gen:builder` target. Each emitted decl's
// Target is filename-only — the router fills Target.Dir from the
// source's directory at the routing phase.
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
				Filename:   strings.ToLower(s.Name) + FilenameSuffix,
				Package:    srcPkg.Name,
				ImportPath: srcPkg.Path,
			}
			// Same-package elision via [emit.Target.ImportPath]
			// renders this reference bare — no self-import on the
			// rendered file.
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
var errAddPackage = errors.New("buildergen: add package to store")

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
//
// srcRef is the reference the emitted `Build` method's return uses
// for the source type — callers in centralised layout pass
// [emit.External] so the renderer adds the import; callers in
// alongside-source layout pass [emit.Builtin] so the renderer emits
// a bare identifier without a self-import.
//
// The Origin of the emitted struct is set to src so the router can
// resolve Target.Dir from the source's file path when target.Dir
// is empty (alongside-source layout).
func (p *Plugin) emitOne(pkg *builder.PackageBuilder, src *node.Struct, target emit.Target, srcRef emit.Ref) {
	builderName := src.Name + p.opts.Suffix
	exported := exportedFields(src)

	pkg.Struct(builderName, func(b *builder.StructBuilder) {
		b.Target(target)
		b.Node().OriginNode = src
		b.Docs(
			builderName + " accumulates field values for " + src.Name +
				" and produces the populated value via " + builderName + ".Build.",
		)
		for _, f := range exported {
			b.Field(unexport(f.Name), refconv.FromNode(f.Type), nil)
		}
		recv := func() emit.Ref { return emit.Ptr(emit.Internal(b.Node())) }
		for _, f := range exported {
			fieldName := f.Name
			fieldType := f.Type
			b.Method("With"+fieldName, func(m *builder.MethodBuilder) {
				m.Receiver("b", recv())
				m.Param("value", refconv.FromNode(fieldType))
				m.Return(recv())
				m.Body(
					emit.NewAssign(
						[]*emit.Expr{emit.NewField(emit.NewIdent("b"), unexport(fieldName))},
						"=",
						[]*emit.Expr{emit.NewIdent("value")},
					),
					emit.NewReturn(emit.NewIdent("b")),
				)
			})
		}
		b.Method("Build", func(m *builder.MethodBuilder) {
			m.Receiver("b", recv())
			m.Return(emit.Ptr(srcRef))
			m.Body(emit.NewReturn(emit.NewAddr(buildComposite(srcRef, exported))))
		})
	})
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

// buildComposite returns the keyed composite-literal expression the
// `Build` method's return statement constructs. Each exported field
// maps to `<FieldName>: b.<unexportedFieldName>`. The composite's
// type renders through the supplied srcRef (a [emit.Builtin] in
// alongside-source layout — bare identifier — or an
// [emit.ExternalRef] in centralised layout — package-qualified).
func buildComposite(srcRef emit.Ref, exported []*node.Field) *emit.Expr {
	keys := make([]string, 0, len(exported))
	vals := make([]*emit.Expr, 0, len(exported))
	for _, f := range exported {
		keys = append(keys, f.Name)
		vals = append(vals, emit.NewField(emit.NewIdent("b"), unexport(f.Name)))
	}
	return emit.NewCompositeKeyed(srcRef, keys, vals)
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
