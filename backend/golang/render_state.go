// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"bytes"
	"errors"
	"fmt"
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
		"render":       s.render,
		"renderType":   s.renderType,
		"renderDocs":   renderDocs,
		"renderFields": s.renderFields,
		"imp":          s.imports.Imp,
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
// Per-field line shape: `\t<Name> <renderType(.Type)>[ ` + "`" + `<.Tag>` + "`" + `][ // <.LineComment>]`.
// DocLines render above each field via [renderDocs] (with the same
// "//"-prefixed-lines-pass-through" semantics used at decl level).
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
		if f.Tag != "" {
			b.WriteString(" `")
			b.WriteString(f.Tag)
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
	default:
		return "", fmt.Errorf("%w: %T", ErrUnsupportedRef, r)
	}
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
