// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package enum is the production multi-output generator for
// idiomatic enum types — a source [node.Enum] (e.g. a Go
// `type Status int` plus its grouped const variants) drives the
// generation of two files: an `_enum.go` carrying the production
// API surface (`String`, `Parse<Type>`, `MarshalJSON`,
// `UnmarshalJSON`) and an `_enum_test.go` carrying the
// round-trip tests that pin the API's contract.
//
// # Source detection
//
// The plugin opts in on each source [node.Enum] carrying a
// `+gen:enum` directive. The enum's [node.EnumVariant] list
// supplies the rendered string-form for each variant, with two
// resolution layers stacked low-to-high:
//
//  1. Default: the variant's [node.EnumVariant.Name] with the
//     enum's [node.Enum.Name] prefix stripped when present and
//     [Options.StripPrefix] is true (the default). So
//     `StatusActive` on `type Status int` renders as the string
//     `"Active"`.
//  2. Override: a `+gen:value <override>` directive on the
//     variant pins the rendered string verbatim, regardless of
//     [Options.StripPrefix]. Source authors reach for the
//     override when the default-derived string clashes with an
//     external protocol's spelling (e.g. snake_case from a JSON
//     API).
//
// # Output set
//
// Two outputs flow from one source enum:
//
//   - Primary `_enum.go`: hosts the [API] kind, rendered by the
//     `enum.api` template. Carries the full production surface.
//   - Tagged "test" `_enum_test.go`: hosts the [Tests] kind,
//     rendered by the `enum.test` template. The `_test.go`
//     suffix triggers the framework's automatic `<pkg>_test`
//     package shift so the tests live in an external test
//     package and can't accidentally read private state.
//
// Single source struct → both files. The Layout phase resolves
// each output's [emit.Target] through the per-output routing
// pipeline so directive / CLI / project-config overrides apply
// per output independently.
//
// # Imports
//
// The plugin expresses every stdlib reference through
// [emit.External] expressions ([API.Sprintf], [API.ErrorsNew],
// etc.). The Go backend's `renderExpr` funcmap registers each
// referenced package on the rendered file's import set
// automatically — the templates carry no hard-coded import
// statements.
package enum

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/core/kind"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/sdk"
)

// Name is the plugin's stable identifier.
const Name = "enum"

// Capability is the capability label the plugin advertises so
// downstream consumers can declare `Requires: []string{Capability}`
// to document a dependency on enum-API generation.
const Capability = "enum"

// DirectiveName is the bare directive name (without the `+gen:`
// or `-gen:` prefix) the plugin reads from source enum types.
const DirectiveName sdk.DirectiveName = "enum"

// Language is the target backend language whose template tree the
// plugin ships. Other languages get the empty signal from
// [Plugin.Outputs] and [Plugin.Templates].
const Language = "golang"

// PrimarySuffix is the per-source-basename suffix the routing
// layer composes for the production output file.
const PrimarySuffix = "_enum.go"

// TestSuffix is the per-source-basename suffix the routing layer
// composes for the test output file. The `_test.go` ending
// triggers the framework's automatic `<pkg>_test` package shift.
const TestSuffix = "_enum_test.go"

// TestOutputTag is the [plugin.Output.Tag] the tagged test
// output carries. Source-side `+gen:out tag=test …` directives
// and CLI `-o enum:test=…` overrides match against this value.
const TestOutputTag = "test"

// SlotName is the [emit.File] slot the plugin appends its
// rendered [API] / [Tests] emit values into. `top` renders the
// content between the package clause and the first core decl —
// the natural placement for whole methods + functions emitted as
// a single template-rendered block.
const SlotName = "top"

// KindAPI is the plugin-defined [emit.Node.Kind] every [API] emit
// value reports. The matching template lives at
// `templates/golang/enum.api.tmpl`.
const KindAPI kind.Kind = "enum.api"

// KindTests is the plugin-defined [emit.Node.Kind] every [Tests]
// emit value reports. The matching template lives at
// `templates/golang/enum.test.tmpl`.
const KindTests kind.Kind = "enum.test"

// DefaultParsePrefix is the prefix the plugin appends to the
// enum's type name to form the parse function's identifier when
// [Options.ParsePrefix] is unset. `ParseStatus` for `type Status
// int` matches the canonical Go idiom.
const DefaultParsePrefix = "Parse"

// DefaultSentinelPrefix is the prefix the plugin appends to the
// enum's type name to form the parse-error sentinel's identifier
// when [Options.SentinelPrefix] is unset. `ErrUnknownStatus`
// reads as a typed sentinel callers compare via [errors.Is].
const DefaultSentinelPrefix = "ErrUnknown"

// ErrEnumHasNoVariants is recorded as a diagnostic when an
// annotated source enum carries no [node.EnumVariant] entries —
// the plugin has nothing to render against. The run continues;
// the offending enum is skipped.
var ErrEnumHasNoVariants = errors.New("enum: source enum has no variants")

// Options carries the plugin's user-tunable settings.
type Options struct {
	// StripPrefix toggles the default name-to-string resolution
	// rule: when true (the default), a variant whose [node.EnumVariant.Name]
	// starts with the enum's [node.Enum.Name] renders with the
	// prefix stripped (e.g. `StatusActive` → `"Active"`). When
	// false, the variant's full name is used verbatim. The
	// `+gen:value <override>` directive on a variant overrides
	// both branches.
	StripPrefix bool `eidos:"strip_prefix,default=true"`

	// ParsePrefix is the prefix the plugin uses to form the parse
	// function's identifier — defaults to [DefaultParsePrefix].
	// Combined with the enum's type name (`ParseStatus`).
	ParsePrefix string `eidos:"parse_prefix,default=Parse"`

	// SentinelPrefix is the prefix the plugin uses to form the
	// parse-error sentinel's identifier — defaults to
	// [DefaultSentinelPrefix]. Combined with the enum's type name
	// (`ErrUnknownStatus`).
	SentinelPrefix string `eidos:"sentinel_prefix,default=ErrUnknown"`
}

// Plugin is the enum generator. Zero value is unusable; go
// through [New] so the embedded [opt.Holder] binds to the
// options field.
type Plugin struct {
	*opt.Holder[Options]
	opts Options
}

// New returns a fresh plugin instance with the options holder
// bound. The pipeline overlays caller-supplied option values via
// [Plugin.SetOptions] (promoted from [opt.Holder]) at Build
// time.
func New() *Plugin {
	p := &Plugin{}
	p.Holder = opt.Bind(&p.opts)
	return p
}

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Priority places the plugin in the foundation generator bucket.
// Foundation runs before composition / cross-cutting buckets so
// downstream plugins discovering the emitted API can read it
// during their own Generate pass.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorFoundation }

// Provides advertises [Capability] so consumers can declare a
// documentary dependency on enum-API generation through their
// own `Requires` list.
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — the enum plugin reads only source
// nodes and has no upstream plugin dependency.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `+gen:enum` schema on source enum
// types.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			On(node.KindEnum).
			Describe("Opts the host enum type in for String / Parse / MarshalJSON / UnmarshalJSON generation.").
			Build(),
	}
}

// Outputs returns the production + test output set in Go. Other
// languages return nil — the plugin ships only golang templates
// today. The Layout phase reads this slice to dispatch each
// emit value to its declared output's filename suffix.
func (*Plugin) Outputs(lang string) []sdk.Output {
	if lang != Language {
		return nil
	}
	return []sdk.Output{
		{Suffix: PrimarySuffix},
		{Tag: TestOutputTag, Suffix: TestSuffix},
	}
}

//go:embed templates/golang/*.tmpl
var templatesFS embed.FS

// Templates returns the embedded template filesystem for the
// requested language. Only `golang` is shipped today; other
// languages receive the empty signal so the backend skips the
// plugin's template-registration pass for that language.
func (*Plugin) Templates(lang string) (fs.FS, bool) {
	if lang != Language {
		return nil, false
	}
	sub, _ := fs.Sub(templatesFS, "templates/"+lang)
	return sub, true
}

// TemplateFuncs returns nil — the plugin ships no funcmap
// extensions; the rendering relies on the backend's canonical
// renderExpr / renderType / renderStmt entries.
func (*Plugin) TemplateFuncs(string) template.FuncMap { return nil }

// TemplateOverrides returns nil — the plugin ships no funcmap
// overrides.
func (*Plugin) TemplateOverrides(string) template.FuncMap { return nil }

// Variant is one rendered enum variant — the source-side
// const identifier plus the resolved string form the rendered
// API maps it to (via `String` / `Parse` / JSON).
type Variant struct {
	// ConstName is the source-side const identifier (e.g.
	// `StatusActive`).
	ConstName string

	// StringValue is the rendered string form — either the
	// auto-derived name (Name minus the enum's type prefix when
	// [Options.StripPrefix] is true) or the `+gen:value`
	// override.
	StringValue string
}

// API is the plugin-defined emit kind every production-output
// emit value reports. The matching `enum.api.tmpl` template
// renders it as the full `String` / `Parse` / `MarshalJSON` /
// `UnmarshalJSON` surface for one source enum.
//
// Every stdlib reference is expressed through [emit.External]
// expressions so the backend's `renderExpr` registers the
// import on the host file's import set automatically — the
// template carries no hard-coded import statements.
type API struct {
	emit.BaseEmit

	// TypeName is the source enum's type identifier (`Status`).
	TypeName string

	// ParseName is the parse function's identifier (`ParseStatus`).
	ParseName string

	// SentinelName is the parse-error sentinel's identifier
	// (`ErrUnknownStatus`).
	SentinelName string

	// Variants is the ordered variant list — declaration order in
	// the source enum, so iota-based numeric values stay aligned
	// with the rendered switch cases.
	Variants []Variant

	// SprintfRef is the `fmt.Sprintf` callee expression — used by
	// the `String` method's fall-through branch when no variant
	// matches. The renderExpr funcmap registers the `fmt` import
	// on the host file.
	SprintfRef *emit.Expr

	// ErrorsNewRef is the `errors.New` callee expression — used
	// to construct the parse-error sentinel.
	ErrorsNewRef *emit.Expr

	// JSONMarshalRef is the `json.Marshal` callee expression —
	// used inside `MarshalJSON`.
	JSONMarshalRef *emit.Expr

	// JSONUnmarshalRef is the `json.Unmarshal` callee expression
	// — used inside `UnmarshalJSON`.
	JSONUnmarshalRef *emit.Expr
}

// Kind returns [KindAPI].
func (*API) Kind() kind.Kind { return KindAPI }

// Compile-time confirmation that *API satisfies [emit.Node].
var _ emit.Node = (*API)(nil)

// Tests is the plugin-defined emit kind every test-output emit
// value reports. The matching `enum.test.tmpl` template renders
// it as the round-trip test suite that pins the production API's
// contract — `Test<Type>String_RoundTrip`,
// `TestParse<Type>_Known`, `TestParse<Type>_Unknown`,
// `Test<Type>JSON_RoundTrip`.
type Tests struct {
	emit.BaseEmit

	// TypeName is the source enum's type identifier.
	TypeName string

	// ParseName is the parse function's identifier.
	ParseName string

	// SentinelName is the parse-error sentinel's identifier.
	SentinelName string

	// Variants is the variant list the test cases iterate.
	Variants []Variant

	// JSONMarshalRef is the `json.Marshal` callee expression —
	// used in the JSON round-trip test.
	JSONMarshalRef *emit.Expr

	// JSONUnmarshalRef is the `json.Unmarshal` callee expression
	// — used in the JSON round-trip test.
	JSONUnmarshalRef *emit.Expr

	// ErrorsIsRef is the `errors.Is` callee expression — used
	// in the unknown-variant test to compare against the
	// sentinel.
	ErrorsIsRef *emit.Expr

	// TestingTPtrRef is the `*testing.T` parameter type — used
	// to register the `testing` import on the host file.
	TestingTPtrRef *emit.Expr
}

// Kind returns [KindTests].
func (*Tests) Kind() kind.Kind { return KindTests }

// Compile-time confirmation that *Tests satisfies [emit.Node].
var _ emit.Node = (*Tests)(nil)

// Generate walks every source enum the reader exposes and, for
// each opted-in enum, queues one [API] contribution against the
// primary output and one [Tests] contribution against the
// "test" output. The Layout phase resolves each contribution's
// [emit.Target] via the per-output routing pipeline; the
// rendered files appear alongside the source enum's file by
// default and follow project / CLI overrides otherwise.
//
// Source enums without `+gen:enum` are skipped silently.
// Annotated enums with no [node.EnumVariant] entries surface
// [ErrEnumHasNoVariants] as a positioned diagnostic; the run
// continues and the offending enum drops from the output set.
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := builder.For(Name, emit.Target{})
	for _, e := range ctx.Reader.Enums().Slice() {
		if !e.HasPositiveDirective(DirectiveName) {
			continue
		}
		if len(e.Variants) == 0 {
			ctx.Diag.Errorf(e.Pos(), "%s: enum %q", ErrEnumHasNoVariants.Error(), e.QName())
			continue
		}
		variants := p.collectVariants(e)
		parseName := p.parsePrefix() + e.Name
		sentinelName := p.sentinelPrefix() + e.Name

		api := &API{
			BaseEmit: emit.BaseEmit{
				OriginNode: e,
				SetByName:  c.SetBy(),
				SourcePos:  e.Pos(),
			},
			TypeName:         e.Name,
			ParseName:        parseName,
			SentinelName:     sentinelName,
			Variants:         variants,
			SprintfRef:       emit.NewExternal("fmt", "Sprintf"),
			ErrorsNewRef:     emit.NewExternal("errors", "New"),
			JSONMarshalRef:   emit.NewExternal("encoding/json", "Marshal"),
			JSONUnmarshalRef: emit.NewExternal("encoding/json", "Unmarshal"),
		}
		if err := ctx.Store.Emit().AppendOriginSlot(e, SlotName, api, c.Provenance("enum.api."+e.Name)); err != nil {
			return fmt.Errorf("%s: append api slot: %w", Name, err)
		}

		tests := &Tests{
			BaseEmit: emit.BaseEmit{
				OriginNode:    e,
				SetByName:     c.SetBy(),
				SourcePos:     e.Pos(),
				OutputTagName: TestOutputTag,
			},
			TypeName:         e.Name,
			ParseName:        parseName,
			SentinelName:     sentinelName,
			Variants:         variants,
			JSONMarshalRef:   emit.NewExternal("encoding/json", "Marshal"),
			JSONUnmarshalRef: emit.NewExternal("encoding/json", "Unmarshal"),
			ErrorsIsRef:      emit.NewExternal("errors", "Is"),
			TestingTPtrRef:   emit.NewExternal("testing", "T"),
		}
		if err := ctx.Store.Emit().AppendOriginSlot(e, SlotName, tests, c.Provenance("enum.test."+e.Name)); err != nil {
			return fmt.Errorf("%s: append test slot: %w", Name, err)
		}
	}
	return nil
}

// collectVariants returns the variant list for e with each
// variant's StringValue resolved through the two-layer rule:
// `+gen:value <override>` wins; otherwise the variant's name
// with the enum's type-name prefix stripped when
// [Options.StripPrefix] is true.
func (p *Plugin) collectVariants(e *node.Enum) []Variant {
	out := make([]Variant, 0, len(e.Variants))
	for _, v := range e.Variants {
		out = append(out, Variant{
			ConstName:   v.Name,
			StringValue: p.resolveStringValue(e.Name, v),
		})
	}
	return out
}

// resolveStringValue applies the two-layer rule: a per-variant
// `+gen:value` override wins outright; otherwise the variant's
// Name is returned with the enum's typeName prefix stripped when
// [Options.StripPrefix] is true.
func (p *Plugin) resolveStringValue(typeName string, v *node.EnumVariant) string {
	if override := v.Directive(pipeline.ValueDirective); override != nil && len(override.Args) > 0 {
		return override.Args[0]
	}
	if p.stripPrefix() && strings.HasPrefix(v.Name, typeName) {
		return strings.TrimPrefix(v.Name, typeName)
	}
	return v.Name
}

// stripPrefix returns the configured StripPrefix value, or the
// documented default (true) when the option binder has not yet
// applied a caller-supplied value (e.g. unit tests bypassing
// SetOptions).
func (p *Plugin) stripPrefix() bool {
	return p.opts.StripPrefix
}

// parsePrefix returns the configured ParsePrefix value or the
// documented default.
func (p *Plugin) parsePrefix() string {
	if p.opts.ParsePrefix != "" {
		return p.opts.ParsePrefix
	}
	return DefaultParsePrefix
}

// sentinelPrefix returns the configured SentinelPrefix value or
// the documented default.
func (p *Plugin) sentinelPrefix() string {
	if p.opts.SentinelPrefix != "" {
		return p.opts.SentinelPrefix
	}
	return DefaultSentinelPrefix
}
