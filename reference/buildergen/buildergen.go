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
// # Output routing
//
// buildergen owns no routing configuration of its own — the
// framework's routing layer composes every emit decl's
// [emit.Target] from the source struct's origin plus the project
// / per-plugin output config and CLI overrides. The plugin
// declares its filename suffix via [Plugin.FilenameSuffix] and
// sets [emit.BaseEmit.OriginNode] on every emit decl; the
// Layout phase does the rest.
package buildergen

import (
	"errors"
	"strings"
	"unicode"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
	"go.thesmos.sh/eidos/reference/internal/refconv"
)

// goNameKey is the cross-language bridge-stamped `go.name` meta
// key. Bridge annotators (the protogo bridge for proto→Go,
// future protorust / prototypescript variants) stamp the
// Go-side rendered identifier on each source field; this plugin
// consults the stamp through [effectiveFieldName] so generated
// setters, composite keys, and the export filter all operate on
// the rendered Go form rather than the source-language form.
// [meta.EnsureKey] resolves to the same registry singleton
// regardless of declaration site, so the plugin shares the key
// with the backend's render-site rule and any other consumer.
//
//nolint:gochecknoglobals // cross-package registry-singleton key
var goNameKey = meta.EnsureKey("go.name", meta.StringParser)

// effectiveFieldName returns the Go-rendered identifier for f.
// A bridge-stamped `go.name` on the field's meta bag wins;
// absent any stamp (Go-source pipelines) the function falls back
// to f.Name verbatim. The helper is the single substitution
// site every f.Name reference inside buildergen routes through
// so the export check, the setter method name, the builder's
// internal field identifier, and the composite-literal key all
// agree on the rendered form.
func effectiveFieldName(f *node.Field) string {
	if got, ok := goNameKey.Get(f.Meta()); ok && got != "" {
		return got
	}
	return f.Name
}

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

// FilenameSuffix is appended to the source-file basename (without
// the `.go` extension) to form the alongside-source output
// filename: `<src-file>_builder.go`. The stringer-style convention
// means every `+gen:builder` struct declared in `article.go`
// composes into a single `article_builder.go`.
const FilenameSuffix = "_builder.go"

// Language is the backend language whose suffix
// [Plugin.FilenameSuffix] returns. Other languages get the empty
// signal until matching templates and per-language suffix
// configuration land.
const Language = "golang"

// Options carries the plugin's user-tunable settings. Routing is
// owned by the framework's routing layer; buildergen exposes no
// output-package option — project / per-plugin output config
// and the CLI `-layout` / `-p` / `-output-dir` flags drive
// where rendered output lands.
type Options struct {
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

// FilenameSuffix returns the per-source filename suffix the Layout
// phase appends to the source's basename when composing the
// rendered output path for every decl this plugin emits.
// Implements [plugin.FilenameProvider]. Returns the empty string
// for backends targeting any language other than [Language] so
// the Layout phase surfaces a missing-FilenameProvider error
// rather than producing a Go-shaped suffix that wouldn't match
// the rendered output.
func (*Plugin) FilenameSuffix(lang string) string {
	if lang == Language {
		return FilenameSuffix
	}
	return ""
}

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

// Generate walks the scoped source-struct bucket and emits a
// builder type for every `+gen:builder` target. Suppression via
// `-gen:builder` skips the source struct even when other generators
// act on it.
//
// Iteration goes through `ctx.Reader.Structs()` so the pipeline's
// scope predicate is honoured at the iterator level — a
// `-target X` run sees only X-shaped source structs.
//
// Routing is owned entirely by the framework's routing layer:
// each emit decl carries Origin set to the source struct and
// [emit.Target] zero. The Layout phase composes Target.Dir /
// Filename / Package / ImportPath from the origin and the
// resolved [pipeline.LayoutPolicy] downstream.
func (p *Plugin) Generate(ctx *plugin.GeneratorContext) error {
	groups, order := groupByPackage(ctx, p.shouldEmit)
	for _, path := range order {
		srcPkg, ok := ctx.Reader.Store().Nodes().Packages().ByQName(path)
		if !ok {
			continue
		}
		c := builder.For(Name, emit.Target{})
		pkg := c.Package(srcPkg.Name, srcPkg.Path)
		for _, s := range groups[path] {
			p.emitOne(pkg, s, emit.External(s.Package, s.Name))
		}
		out, err := pkg.Build()
		if err != nil {
			return err
		}
		if err := ctx.Store.Emit().AddPackage(out); err != nil {
			return errors.Join(errAddPackage, err)
		}
	}
	return nil
}

// errAddPackage is the sentinel wrapped around store-side AddPackage
// failures. Tests and callers detect the class with errors.Is.
//
//nolint:gochecknoglobals // sentinel.
var errAddPackage = errors.New("buildergen: add package to store")

// groupByPackage walks the scoped source-struct bucket and groups
// every match of pred by source-package import path. The returned
// order slice preserves first-encountered path order so iteration
// of the grouping stays deterministic across runs.
func groupByPackage(
	ctx *plugin.GeneratorContext,
	pred func(*node.Struct) bool,
) (map[string][]*node.Struct, []string) {
	groups := map[string][]*node.Struct{}
	order := []string{}
	for _, s := range ctx.Reader.Structs().Slice() {
		if !pred(s) {
			continue
		}
		if _, seen := groups[s.Package]; !seen {
			order = append(order, s.Package)
		}
		groups[s.Package] = append(groups[s.Package], s)
	}
	return groups, order
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
// srcRefBase is the reference the emitted `Build` method's return
// uses for the source type. The plugin always passes
// [emit.External]; the renderer elides self-imports for
// same-package references via the Target.ImportPath the Layout
// phase composes from the source package, so the qualified form
// stays correct under both alongside-source and centralised
// layouts.
//
// Generic source structs propagate their type parameters to the
// emitted builder verbatim — `Dispatcher[T any]` produces
// `DispatcherBuilder[T any]` with methods whose receivers and
// returns carry `[T]` and Build returning `*Dispatcher[T]`. The
// type-arg list is threaded through receiver / return / composite
// references via `typeArgsFromParams`.
//
// The Origin of the emitted struct is set to src so the Layout
// phase can resolve every Target field downstream — the plugin
// itself never constructs an [emit.Target] literal.
func (p *Plugin) emitOne(pkg *builder.PackageBuilder, src *node.Struct, srcRefBase emit.Ref) {
	builderName := src.Name + p.opts.Suffix
	exported := exportedFields(src)
	typeArgs := builder.TypeArgsFromNodeParams(src.TypeParams)
	srcRef := builder.ApplyTypeArgs(srcRefBase, typeArgs)

	pkg.Struct(builderName, func(b *builder.StructBuilder) {
		b.Origin(src)
		b.Docs(
			builderName + " accumulates field values for " + src.Name +
				" and produces the populated value via " + builderName + ".Build.",
		)
		for _, tp := range src.TypeParams {
			b.TypeParam(tp.Name, refconv.ConstraintFromNode(tp.Constraint))
		}
		for _, f := range exported {
			// The builder's internal field is a derived holder of the
			// accumulated source value, not a projection of the source
			// field itself. Threading Origin on the [emit.Field] would
			// route the rendered field name through the bridge-aware
			// [fieldNameFor] rule, substituting the source field's
			// translated identifier for the builder's intentionally-
			// unexported `fieldIdent(...)` form — a mismatch with
			// the setter body that assigns to that identifier. Type
			// provenance still flows through [refconv.FromNode], which
			// threads the source TypeRef's OriginNode so the bridge's
			// `go.type` override resolves at render time.
			b.Field(fieldIdent(effectiveFieldName(f)), refconv.FromNode(f.Type), nil)
		}
		recv := func() emit.Ref { return emit.Ptr(emit.Internal(b.Node(), typeArgs...)) }
		for _, f := range exported {
			fieldName := effectiveFieldName(f)
			fieldType := f.Type
			b.Method("With"+fieldName, func(m *builder.MethodBuilder) {
				m.Receiver("b", recv())
				m.Param("value", refconv.FromNode(fieldType))
				m.Return(recv())
				m.Body(
					emit.NewAssign(
						[]*emit.Expr{emit.NewField(emit.NewIdent("b"), fieldIdent(fieldName))},
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

// fieldIdent returns the unexported identifier the builder uses for
// the field of the supplied source name, with a trailing underscore
// appended when the unexported form would collide with a Go
// keyword (`type`, `default`, `range`, …). The keyword set is the
// Go specification's reserved word list; identifiers outside that
// set pass through [unexport] unchanged.
func fieldIdent(name string) string {
	id := unexport(name)
	if isGoKeyword(id) {
		return id + "_"
	}
	return id
}

// goKeywords is the set of Go reserved words that may not appear as
// identifiers. Lookup is O(1) via map indexing.
//
//nolint:gochecknoglobals // set literal used as a read-only lookup table.
var goKeywords = map[string]struct{}{
	"break": {}, "case": {}, "chan": {}, "const": {}, "continue": {},
	"default": {}, "defer": {}, "else": {}, "fallthrough": {}, "for": {},
	"func": {}, "go": {}, "goto": {}, "if": {}, "import": {},
	"interface": {}, "map": {}, "package": {}, "range": {}, "return": {},
	"select": {}, "struct": {}, "switch": {}, "type": {}, "var": {},
}

// isGoKeyword reports whether id is a Go reserved word.
func isGoKeyword(id string) bool {
	_, ok := goKeywords[id]
	return ok
}

// exportedFields returns the exported fields of src in declaration
// order. Unexported fields are skipped — the emitted `With<Field>`
// surface mirrors what a user-authored builder for the type could
// reach.
//
// The export check runs against [effectiveFieldName] so proto-
// source fields with the bridge-stamped PascalCase Go identifier
// pass the same upper-case-first rule a Go-source PascalCase
// field would. Without the substitution, snake_case proto field
// names would be filtered out and the rendered builder would
// carry no setters.
func exportedFields(s *node.Struct) []*node.Field {
	return s.FieldsWith(func(f *node.Field) bool {
		name := effectiveFieldName(f)
		return name != "" && unicode.IsUpper(rune(name[0]))
	})
}

// buildComposite returns the keyed composite-literal expression the
// `Build` method's return statement constructs. Each exported field
// maps to `<FieldName>: b.<builderFieldIdent>`, where
// `<builderFieldIdent>` is [fieldIdent] of the rendered Go-side
// name — the same keyword-safe identifier the builder struct
// declares for its accumulating field. The composite's type
// renders through the supplied srcRef so generic instantiations
// carry their type-arg list through Build.
//
// The composite key is the rendered Go-side name resolved via
// [effectiveFieldName] so proto-source fields land on the Go
// struct's PascalCase field rather than the source's snake_case
// form.
func buildComposite(srcRef emit.Ref, exported []*node.Field) *emit.Expr {
	keys := make([]string, 0, len(exported))
	vals := make([]*emit.Expr, 0, len(exported))
	for _, f := range exported {
		name := effectiveFieldName(f)
		keys = append(keys, name)
		vals = append(vals, emit.NewField(emit.NewIdent("b"), fieldIdent(name)))
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
