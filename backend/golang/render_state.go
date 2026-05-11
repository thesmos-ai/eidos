// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/writer"
)

// ErrTemplateMissing is returned by render-dispatch helpers when no
// template is registered for an entity's [emit.Node.Kind]. The
// wrapping error names the offending kind so the diagnostic is
// actionable without a stack trace.
var ErrTemplateMissing = errors.New("backend/golang: no template registered for kind")

// ErrUnsupportedRef is returned by [renderState.renderType] when
// called with an [emit.Ref] kind the current funcmap can't render,
// or by [internalTargetName] when a [emit.TypeRef] points at a
// target kind whose name can't yet be extracted. The wrapped
// message names the concrete Go type so diagnostics attribute the
// gap precisely.
var ErrUnsupportedRef = errors.New("backend/golang: unsupported Ref")

// ErrUnsupportedExpr is returned by [renderExpr] when called with
// an [emit.ExprKind] or [emit.LiteralKind] variant the current
// funcmap can't render. The expression-rendering surface widens
// as additional variants are wired in; the wrapped message names
// the offending kind so diagnostics attribute the gap precisely.
var ErrUnsupportedExpr = errors.New("backend/golang: unsupported Expr")

// ErrMixedNamedParams is returned by [renderState.renderParams]
// when called with a parameter list that mixes named and unnamed
// entries — forbidden by Go's grammar ("Within a list of
// parameters or results, the names must either all be present or
// all be absent"). The wrapped message names the offending entity
// so generators can locate and fix the inconsistency.
var ErrMixedNamedParams = errors.New("backend/golang: param list mixes named and unnamed entries")

// renderState carries the per-Target rendering context: a cloned
// template tree with funcmap closures bound to this state, and a
// fresh [writer.ImportSet] that accumulates every `imp` call the
// templates make during render.
//
// The clone-per-Target pattern is the package's foundational
// race-freedom property. Targets render independently — concurrent
// dispatch is safe by construction:
//
//   - The parent template tree (parsed once via [loadTemplates]) is
//     immutable from the moment construction completes; it carries
//     parse-time placeholder funcs so the parser's name-and-arity
//     validation succeeds.
//   - Each Target render gets its own [template.Template.Clone] of
//     the parent tree. Clones share the parse tree (cheap) but own
//     their funcmap namespace.
//   - The funcmap attached to the clone is built from this state
//     struct; the closures bind `s` rather than reading any global,
//     so per-Target import tracking is naturally isolated.
//
// This is the extension point plugin-template merge (later phases)
// builds on: plugin-supplied templates and funcmap entries layer on
// top of the per-Target clone, never the parent. The parent stays
// the deterministic, immutable canonical state.
type renderState struct {
	tmpl    *template.Template
	imports *writer.ImportSet
}

// newRenderState clones root, attaches a fresh funcmap whose
// closures bind the returned state, and returns the ready-to-execute
// rendering context for one Target.
func newRenderState(root *template.Template) *renderState {
	clone := template.Must(root.Clone())
	s := &renderState{
		tmpl:    clone,
		imports: writer.NewImportSet(nil),
	}
	clone.Funcs(s.funcMap())
	return s
}

// funcMap returns the canonical core funcmap with every closure
// bound to s. The reserved-name set surfaced here covers the
// render-dispatch and import-collection families; plugin overrides
// for these names are rejected at Build time.
func (s *renderState) funcMap() template.FuncMap {
	return template.FuncMap{
		"render":           s.render,
		"renderType":       s.renderType,
		"renderDocs":       renderDocs,
		"renderFields":     s.renderFields,
		"renderEmbeds":     s.renderEmbeds,
		"renderTypeParams": s.renderTypeParams,
		"renderParams":     s.renderParams,
		"renderReturns":    s.renderReturns,
		"renderExpr":       renderExpr,
		"imp":              s.imports.Imp,
	}
}

// renderDocs converts a doc-comment-line slice into the canonical
// Go-comment-block text rendered above a declaration. Each line is
// prefixed with "// " unless it already begins with "//" — typically
// a compile-time directive such as "//go:embed", "//go:build", or
// "//nolint:foo" — in which case the line renders verbatim. Empty
// input returns the empty string so callers can place it directly
// above a declaration without introducing whitespace for
// undocumented entities.
//
// Generators that mix human docs and directive lines place the
// directives last so the rendered ordering is "docs first, then
// directives, then declaration" per Go convention.
//
// `renderDocs` is one of the reserved core funcmap entries — plugin
// overrides for this name are rejected at Build time.
func renderDocs(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "//") {
			b.WriteString(line)
		} else {
			b.WriteString("// ")
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// render executes the template named after n's [emit.Node.Kind] and
// returns the rendered text inline. Returns [ErrTemplateMissing]
// wrapped with the kind when no template is registered.
func (s *renderState) render(n emit.Node) (string, error) {
	kind := string(n.Kind())
	if s.tmpl.Lookup(kind) == nil {
		return "", fmt.Errorf("%w: %s", ErrTemplateMissing, kind)
	}
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, kind, n); err != nil {
		return "", fmt.Errorf("backend/golang: render %s: %w", kind, err)
	}
	return buf.String(), nil
}

// renderFields produces the body content of a Go struct
// declaration — every field rendered on its own tab-indented line,
// separated by blank lines on the doc-boundary so each documented
// field forms a visually grouped unit per Go convention.
//
// Per-field line shape: `\t<Name> <renderType(.Type)>[ ` + "`" + `<tags>` + "`" + `][ // <.LineComment>]`.
// DocLines render above each field via [renderDocs] (with the same
// "//"-prefixed-lines-pass-through" semantics used at decl level).
//
// Tag aggregation: the rendered tag blob unions [emit.Field.Tag]
// (the host generator's base tag, rendered verbatim) with every
// [*emit.Tag] appended to [emit.Field.Tags] (cross-cutting plugin
// contributions). Slot entries render as `Key:"EscapedValue"` —
// values are Go-string-escaped via [strconv.Quote]. Order: base
// tag first, then slot entries in append order. Both empty produces
// no tag at all (not even the backticks).
//
// Separation rule: emit a blank line BETWEEN any two adjacent
// fields when either carries DocLines — this groups documented
// fields with their docs and visually separates them from
// undocumented siblings, without introducing a trailing blank
// before the closing brace.
//
// `renderFields` is one of the reserved canonical-render funcmap
// entries — plugin overrides are rejected at Build time.
func (s *renderState) renderFields(fields []*emit.Field) (string, error) {
	var b strings.Builder
	for i, f := range fields {
		if i > 0 && (len(f.DocLines) > 0 || len(fields[i-1].DocLines) > 0) {
			b.WriteByte('\n')
		}
		b.WriteString(renderDocs(f.DocLines))
		b.WriteByte('\t')
		b.WriteString(f.Name)
		b.WriteByte(' ')
		t, err := s.renderType(f.Type)
		if err != nil {
			return "", err
		}
		b.WriteString(t)
		if tag := fieldTagBlob(f); tag != "" {
			b.WriteString(" `")
			b.WriteString(tag)
			b.WriteByte('`')
		}
		if f.LineComment != "" {
			b.WriteString(" // ")
			b.WriteString(f.LineComment)
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// fieldTagBlob assembles the backtick-wrapped struct-tag content
// for a single field. The base [emit.Field.Tag] (raw text the host
// generator declared) renders first, followed by each [*emit.Tag]
// appended to the field's tags slot in append order; non-Tag slot
// entries are ignored so the helper degrades gracefully when a
// plugin appends an unrelated node by mistake. Returns the empty
// string when neither the base tag nor the slot carry content; the
// caller omits the backtick wrap entirely in that case.
func fieldTagBlob(f *emit.Field) string {
	var b strings.Builder
	if f.Tag != "" {
		b.WriteString(f.Tag)
	}
	for _, item := range f.Tags().Items {
		t, ok := item.(*emit.Tag)
		if !ok {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(t.Key)
		b.WriteByte(':')
		b.WriteString(strconv.Quote(t.Value))
	}
	return b.String()
}

// renderEmbeds produces the rendered embed lines that go inside a
// struct or interface body — one tab-indented `renderType(.Type)`
// per embed, terminated by a newline. Empty input returns the
// empty string; the caller emits no leading or trailing whitespace
// for missing embeds. Currently used by the `emit.struct` and
// `emit.interface` templates.
func (s *renderState) renderEmbeds(embeds []*emit.Embed) (string, error) {
	var b strings.Builder
	for _, e := range embeds {
		t, err := s.renderType(e.Type)
		if err != nil {
			return "", err
		}
		b.WriteByte('\t')
		b.WriteString(t)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// renderTypeParams produces the bracketed generic-parameter list of
// a Go declaration — `[T1 C1, T2 C2]` joined by ", ". Each entry is
// `Name renderType(<bound>)` for type parameters with an explicit
// constraint, falling back to `Name any` for [emit.Constraint.IsAny]
// parameters (preserving Go's grammar requirement that every
// parameter declare a bound).
//
// A constraint with a single [emit.Constraint.Embedded] ref
// renders the ref directly; multiple embeds render as an inline
// interface (`{ E1; E2 }`) — the intersection form Go allows for
// composite constraints.
//
// Empty input returns the empty string; the caller emits no
// brackets when the declaration has no type parameters. The helper
// is reserved canonical-render.
func (s *renderState) renderTypeParams(params []*emit.TypeParam) (string, error) {
	if len(params) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		entry, err := s.renderTypeParam(p)
		if err != nil {
			return "", err
		}
		parts = append(parts, entry)
	}
	return "[" + strings.Join(parts, ", ") + "]", nil
}

// renderTypeParam renders a single type-parameter entry as it
// appears in the bracket list — name + space + rendered bound.
// Unconstrained parameters (nil or `IsAny` Constraint) render
// with the predeclared `any` constraint.
func (s *renderState) renderTypeParam(p *emit.TypeParam) (string, error) {
	if p.Constraint.IsAny() {
		return p.Name + " any", nil
	}
	embedded := p.Constraint.Embedded
	if len(embedded) == 1 {
		bound, err := s.renderType(embedded[0])
		if err != nil {
			return "", err
		}
		return p.Name + " " + bound, nil
	}
	parts := make([]string, 0, len(embedded))
	for _, e := range embedded {
		r, err := s.renderType(e)
		if err != nil {
			return "", err
		}
		parts = append(parts, r)
	}
	return p.Name + " interface { " + strings.Join(parts, "; ") + " }", nil
}

// renderParams produces the parenthesised parameter list of a Go
// function or method signature. Each entry is `Name renderType(Type)`
// when names are present and `renderType(Type)` when names are
// absent. Variadic parameters receive the `...` prefix on their
// type. An empty parameter list renders as `()`.
//
// Mixed-named parameters (a list where some entries have names and
// others don't) violate Go's grammar; this case fails with
// [ErrMixedNamedParams] wrapped with the parameter-name context.
//
// `renderParams` is one of the reserved canonical-render funcmap
// entries — plugin overrides are rejected at Build time.
func (s *renderState) renderParams(params []*emit.Param) (string, error) {
	if len(params) == 0 {
		return "()", nil
	}
	var anyNamed, anyUnnamed bool
	for _, p := range params {
		if p.Name == "" {
			anyUnnamed = true
		} else {
			anyNamed = true
		}
	}
	if anyNamed && anyUnnamed {
		return "", fmt.Errorf("%w: %s", ErrMixedNamedParams, paramListSummary(params))
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		entry, err := s.renderParamEntry(p)
		if err != nil {
			return "", err
		}
		parts = append(parts, entry)
	}
	return "(" + strings.Join(parts, ", ") + ")", nil
}

// renderParamEntry renders a single parameter as it appears inside
// the parenthesised list — name + type (or just type for anonymous
// entries), with `...` prefixing the type for variadic params.
func (s *renderState) renderParamEntry(p *emit.Param) (string, error) {
	t, err := s.renderType(p.Type)
	if err != nil {
		return "", err
	}
	if p.Variadic {
		t = "..." + t
	}
	if p.Name == "" {
		return t, nil
	}
	return p.Name + " " + t, nil
}

// paramListSummary returns a short, comma-separated list of the
// parameter names (using `_` for anonymous entries) suitable for
// inclusion in diagnostic messages.
func paramListSummary(params []*emit.Param) string {
	names := make([]string, 0, len(params))
	for _, p := range params {
		if p.Name == "" {
			names = append(names, "_")
		} else {
			names = append(names, p.Name)
		}
	}
	return strings.Join(names, ", ")
}

// renderReturns produces the return-clause text of a Go function
// or method signature, following the three-case truth table:
//
//   - Zero returns → empty string (no clause).
//   - Exactly one unnamed return → bare `renderType(Type)` (no
//     parentheses).
//   - Multiple returns → parenthesised, comma-separated list.
//
// `renderReturns` is one of the reserved canonical-render funcmap
// entries — plugin overrides are rejected at Build time.
func (s *renderState) renderReturns(returns []emit.Ref) (string, error) {
	switch len(returns) {
	case 0:
		return "", nil
	case 1:
		return s.renderType(returns[0])
	}
	parts := make([]string, 0, len(returns))
	for _, r := range returns {
		t, err := s.renderType(r)
		if err != nil {
			return "", err
		}
		parts = append(parts, t)
	}
	return "(" + strings.Join(parts, ", ") + ")", nil
}

// renderExpr produces the Go source spelling for an [emit.Expr].
// The supported variant surface widens as additional expression
// kinds are wired in; today the function handles literal values
// ([emit.ExprLiteral] across every [emit.LiteralKind]) and bare
// identifiers ([emit.ExprIdent], used for builtin idents like
// "iota"). Any other variant returns [ErrUnsupportedExpr] wrapped
// with the offending kind.
//
// Nil input returns the empty string so callers can place the
// helper directly into templates without explicit nil-guards on
// optional initialisers.
//
// `renderExpr` is one of the reserved dispatch funcmap entries —
// plugin overrides are rejected at Build time.
func renderExpr(e *emit.Expr) (string, error) {
	if e == nil {
		return "", nil
	}
	switch e.ExprKind {
	case emit.ExprLiteral:
		return renderLiteral(e)
	case emit.ExprIdent:
		return e.Name, nil
	default:
		return "", fmt.Errorf("%w: ExprKind=%s", ErrUnsupportedExpr, e.ExprKind)
	}
}

// renderLiteral produces the Go source spelling for a single
// literal expression, dispatching on the [emit.LiteralKind].
// Strings are re-quoted via [strconv.Quote]; numeric and boolean
// literals render their raw text; nil renders as the keyword;
// runes wrap in single quotes; raw literals pass through.
func renderLiteral(e *emit.Expr) (string, error) {
	switch e.LitKind {
	case emit.LitString:
		return strconv.Quote(e.RawText), nil
	case emit.LitInt, emit.LitUint, emit.LitFloat:
		return e.RawText, nil
	case emit.LitBool:
		return e.RawText, nil
	case emit.LitNil:
		return "nil", nil
	case emit.LitRune:
		return "'" + e.RawText + "'", nil
	case emit.LitRaw:
		return e.RawText, nil
	default:
		return "", fmt.Errorf("%w: LitKind=%s", ErrUnsupportedExpr, e.LitKind)
	}
}

// renderType produces the Go source spelling for r. Supported kinds:
//
//   - [emit.BuiltinRef] — rendered as the builtin's Name.
//   - [emit.ExternalRef] — rendered as "<alias>.<Name>", with
//     <alias> obtained by registering the package path via [renderState.imp].
//   - [emit.TypeRef] — rendered as the unqualified name of the
//     target node (TypeRef is same-package by contract).
//
// Other ref kinds return [ErrUnsupportedRef] wrapped with the
// concrete Go type.
func (s *renderState) renderType(r emit.Ref) (string, error) {
	switch typed := r.(type) {
	case *emit.BuiltinRef:
		return typed.Name, nil
	case *emit.ExternalRef:
		alias, err := s.imports.Imp(typed.Package)
		if err != nil {
			return "", fmt.Errorf("backend/golang: renderType: %w", err)
		}
		return alias + "." + typed.Name, nil
	case *emit.TypeRef:
		return internalTargetName(typed.Target)
	case *emit.CompositeRef:
		return s.renderComposite(typed)
	default:
		return "", fmt.Errorf("%w: %T", ErrUnsupportedRef, r)
	}
}

// renderComposite dispatches on the [emit.CompositeRef.Shape] and
// returns the Go source spelling for the composite. The
// supported-shape surface widens phase by phase; today only
// [emit.ShapeUnion] is wired (driven by generic-constraint
// rendering). Other shapes return [ErrUnsupportedRef] wrapped with
// the offending shape.
func (s *renderState) renderComposite(r *emit.CompositeRef) (string, error) {
	switch r.Shape {
	case emit.ShapeUnion:
		return s.renderUnion(r.UnionTerms)
	default:
		return "", fmt.Errorf("%w: composite shape %s", ErrUnsupportedRef, r.Shape)
	}
}

// renderUnion produces the Go union-constraint spelling for a
// `T1 | T2 | ~T3` sequence: terms joined by " | ", with the
// approximation marker `~` prefixing terms whose Approx flag is
// set. Empty term slices yield the empty string — the caller
// (typically [renderTypeParams]) is responsible for treating an
// empty constraint as a programming error if relevant.
func (s *renderState) renderUnion(terms []emit.UnionTerm) (string, error) {
	parts := make([]string, 0, len(terms))
	for _, t := range terms {
		rendered, err := s.renderType(t.Type)
		if err != nil {
			return "", err
		}
		if t.Approx {
			rendered = "~" + rendered
		}
		parts = append(parts, rendered)
	}
	return strings.Join(parts, " | "), nil
}

// internalTargetName returns the unqualified declaration name of a
// [emit.TypeRef] target node — the spelling used when the target
// lives in the same Go package as the referring entity. Returns
// [ErrUnsupportedRef] wrapped with the concrete Go type when the
// target is a kind whose name can't be extracted.
func internalTargetName(n emit.Node) (string, error) {
	switch t := n.(type) {
	case *emit.Struct:
		return t.Name, nil
	case *emit.Interface:
		return t.Name, nil
	case *emit.Alias:
		return t.Name, nil
	case *emit.Enum:
		return t.Name, nil
	case *emit.Function:
		return t.Name, nil
	default:
		return "", fmt.Errorf("%w: TypeRef target %T", ErrUnsupportedRef, n)
	}
}
