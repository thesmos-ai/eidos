// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/writer"
)

// ErrTemplateMissing is returned by render-dispatch helpers when no
// template is registered for an entity's [emit.Node.Kind]. The
// wrapping error names the offending kind so the diagnostic is
// actionable without a stack trace.
var ErrTemplateMissing = errors.New("backend/golang: no template registered for kind")

// renderState carries the per-Target rendering context: a cloned
// template tree with funcmap closures bound to this state, a fresh
// [writer.ImportSet] that accumulates every `imp` call the
// templates make during render, and the plugin topo order that
// drives slot-contribution sequencing.
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

	// pluginOrder lists plugin names in capability-topo order — the
	// resolved sequence the pipeline produces via
	// [plugin.BackendContext.Ordered]. Slot contributions are
	// rendered grouped by this order so the same input emits the
	// same output across runs. Empty when no plugins participated
	// (typical for direct backend tests); contributions then render
	// in raw append order.
	pluginOrder []string
}

// newRenderState clones root, attaches a fresh funcmap whose
// closures bind the returned state, and returns the ready-to-execute
// rendering context for one Target. pluginOrder carries the
// resolved capability-topo plugin sequence used to order slot
// contributions.
func newRenderState(root *template.Template, pluginOrder []string) *renderState {
	clone := template.Must(root.Clone())
	s := &renderState{
		tmpl:        clone,
		imports:     writer.NewImportSet(nil),
		pluginOrder: pluginOrder,
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
		"render":                 s.render,
		"renderType":             s.renderType,
		"renderDocs":             renderDocs,
		"renderTypeParams":       s.renderTypeParams,
		"renderParams":           s.renderParams,
		"renderReceiver":         s.renderReceiver,
		"renderReturns":          s.renderReturns,
		"renderExpr":             s.renderExpr,
		"renderStmt":             s.renderStmt,
		"renderStructFields":     s.renderStructFields,
		"renderStructEmbeds":     s.renderStructEmbeds,
		"renderStructMethods":    s.renderStructMethods,
		"renderInterfaceEmbeds":  s.renderInterfaceEmbeds,
		"renderInterfaceMethods": s.renderInterfaceMethods,
		"renderEnumVariants":     s.renderEnumVariants,
		"renderFunctionBody":     s.renderFunctionBody,
		"renderMethodBody":       s.renderMethodBody,
		"renderFunctionParams":   s.renderFunctionParams,
		"renderMethodParams":     s.renderMethodParams,
		"imp":                    s.imports.Imp,
	}
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
