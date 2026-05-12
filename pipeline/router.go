// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"path/filepath"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/store"
)

// runRouter is the pipeline phase that resolves [emit.Target.Dir]
// and [emit.Target.Filename] on every emit entity in the store
// according to the precedence rule the CLI design contract pins:
//
//  1. Plugin's explicit Target.Dir / Target.Filename win — the
//     escape hatch for synthetic emit that has no source attribution.
//  2. [OutDirective] on the originating source node stamps
//     Target.Filename.
//  3. Default for Target.Dir: the directory of the originating
//     source file (`filepath.Dir(entity.Origin().Pos().File)`). The
//     "alongside-source" layout — every generated file lives next
//     to the source entity it derives from.
//
// Synthetic emit (no Origin, no explicit Target.Dir) is a router
// error: the framework can't know where to place the file without
// either a source anchor or an explicit plugin override. The error
// surfaces as a [diag.Error] attributed to the pipeline's "router"
// channel and the run continues — the offending entity is left
// with an empty Target.Dir, which the disk sink will reject at
// write time so the failure is visible.
//
// The router never modifies Target.Package; package routing stays
// plugin territory. The router skips [emit.File] entities entirely
// — files are composed via `store.EmitView.FileFor(target)`, which
// already pins Dir/Name/Package from the supplied Target.
func (p *Pipeline) runRouter(s *store.Store) {
	ps := p.diag.For("pipeline.router")
	v := s.Emit()

	// Each routable kind has its own field-typed setter — emit
	// doesn't expose a single `WithTarget` interface — so the
	// router walks per-bucket and dispatches accordingly.

	v.Structs().Range(func(e *emit.Struct) bool {
		t, ok := resolveTarget(e.Target, e.Origin())
		if !ok {
			emitRouterError(ps, "struct", e.QName())
			return true
		}
		e.Target = t
		return true
	})

	v.Interfaces().Range(func(e *emit.Interface) bool {
		t, ok := resolveTarget(e.Target, e.Origin())
		if !ok {
			emitRouterError(ps, "interface", e.QName())
			return true
		}
		e.Target = t
		return true
	})

	v.Functions().Range(func(e *emit.Function) bool {
		t, ok := resolveTarget(e.Target, e.Origin())
		if !ok {
			emitRouterError(ps, "function", e.QName())
			return true
		}
		e.Target = t
		return true
	})

	v.Variables().Range(func(e *emit.Variable) bool {
		t, ok := resolveTarget(e.Target, e.Origin())
		if !ok {
			emitRouterError(ps, "variable", e.QName())
			return true
		}
		e.Target = t
		return true
	})

	v.Constants().Range(func(e *emit.Constant) bool {
		t, ok := resolveTarget(e.Target, e.Origin())
		if !ok {
			emitRouterError(ps, "constant", e.QName())
			return true
		}
		e.Target = t
		return true
	})

	v.Enums().Range(func(e *emit.Enum) bool {
		t, ok := resolveTarget(e.Target, e.Origin())
		if !ok {
			emitRouterError(ps, "enum", e.QName())
			return true
		}
		e.Target = t
		return true
	})

	v.Aliases().Range(func(e *emit.Alias) bool {
		// [emit.Alias] stores its file target in [Alias.File] (not
		// Target — that field is the aliased type Ref). Same
		// resolution rule.
		t, ok := resolveTarget(e.File, e.Origin())
		if !ok {
			emitRouterError(ps, "alias", e.QName())
			return true
		}
		e.File = t
		return true
	})
}

// resolveTarget applies the router precedence to a Target and its
// originating source node. Returns the resolved Target and true
// when routing succeeded, or the zero Target and false when the
// entity has neither an explicit Dir nor an Origin to derive from.
//
// When Target.Filename is empty, callers expect the plugin to have
// set it; the router does not synthesise a filename. The Filename
// is only mutated when the originating node carries an [OutDirective].
func resolveTarget(t emit.Target, origin node.Node) (emit.Target, bool) {
	// Filename override via `+gen:out` on the originating node.
	if origin != nil {
		if fn, ok := outDirectiveFilename(origin); ok {
			t.Filename = fn
		}
	}

	if t.Dir != "" {
		return t, true
	}
	// Default: alongside the source file.
	if origin != nil {
		pos := origin.Pos()
		if pos.File != "" {
			t.Dir = filepath.Dir(pos.File)
			return t, true
		}
	}
	return emit.Target{}, false
}

// outDirectiveFilename inspects n's directive list for [OutDirective].
// Returns the first positional argument (the filename) and true when
// present; "" and false otherwise.
func outDirectiveFilename(n node.Node) (string, bool) {
	for _, d := range n.Directives() {
		if d.Name == OutDirective {
			if len(d.Args) > 0 {
				return d.Args[0], true
			}
		}
	}
	return "", false
}

// emitRouterError attaches an Error diagnostic to ps describing an
// unroutable entity. The pipeline continues with the next entity;
// the disk sink rejects the empty-Dir Target at write time so the
// failure surfaces a second time at IO if the run still attempts
// the write.
func emitRouterError(ps *diag.PluginSink, kind, qname string) {
	ps.Errorf(
		position.Pos{},
		"unroutable %s %q: no plugin Target.Dir and no source Origin to derive from",
		kind, qname,
	)
}
