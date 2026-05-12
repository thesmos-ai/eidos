// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package shapewriter detects structs that satisfy the io.Writer
// shape — a method named `Write` taking a `[]byte` and returning
// `(int, error)`. Downstream generators read the stamped metadata
// (`shape.writer.detected`, `shape.writer.method`) to decide
// whether their codegen path applies to a given struct.
//
// Detection is heuristic by default; directive overrides force or
// suppress the detection regardless of the heuristic outcome:
//
//   - `+gen:writer` forces detection on the host struct, even
//     when no method matches the canonical signature.
//   - `-gen:writer` suppresses detection on the host struct, even
//     when the method exists with the matching signature.
package shapewriter

import (
	"fmt"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/priority"
)

// Name is the plugin's stable identifier surfaced through
// [plugin.Plugin.Name] for ordering tie-breaks, diagnostic
// attribution, and cache-key derivation.
const Name = "shape-writer"

// DirectiveName is the bare directive name (without the `+gen:` or
// `-gen:` prefix) the plugin reads from each struct's directive
// list. The positive form forces detection; the negative form
// suppresses it.
const DirectiveName directive.Name = "writer"

// Detected is the meta key the plugin stamps with `true` when the
// struct matches the writer shape and `false` otherwise. The key
// is always set — consumers don't need to distinguish "annotator
// didn't run" from "annotator ran, no match".
//
//nolint:gochecknoglobals // package-level meta key, registered once at init.
var Detected = meta.NewKey("shape.writer.detected", meta.BoolParser)

// MethodQName is the meta key the plugin stamps with the matched
// method's fully-qualified name (`<ownerQName>.<methodName>`) on
// detected structs. Empty when the heuristic does not match — the
// directive-driven positive override sets detected=true without a
// method, in which case the key value is left empty and consumers
// must guard accordingly.
//
//nolint:gochecknoglobals // package-level meta key, registered once at init.
var MethodQName = meta.NewKey("shape.writer.method", meta.StringParser)

// writeMethodName is the unqualified method name the heuristic
// targets — the canonical `Write` slot on io.Writer.
const writeMethodName = "Write"

// Plugin is the writer-shape annotator. The zero value is usable.
type Plugin struct{}

// New returns a ready-to-register plugin. Provided for parity with
// other plugins that need constructor-time wiring.
func New() *Plugin { return &Plugin{} }

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Priority places the plugin in the shape-detector bucket so it
// runs alongside other annotators that stamp `shape.*` metadata.
func (*Plugin) Priority() priority.Priority { return priority.AnnotatorShape }

// Provides returns nil — the plugin doesn't expose a capability
// label for cross-plugin topo ordering. Downstream consumers reach
// the shape metadata directly through the meta keys.
func (*Plugin) Provides() []string { return nil }

// Requires returns nil — the plugin has no upstream dependency.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `+gen:writer` / `-gen:writer` schema with
// the pipeline so directive validation rejects malformed uses at
// frontend-parse time.
func (*Plugin) Directives() []directive.Schema {
	return []directive.Schema{
		directive.NewSchema(DirectiveName).
			On(node.KindStruct).
			Describe("Forces (+) or suppresses (-) writer-shape detection on the host struct.").
			Build(),
	}
}

// Annotate iterates the node store's struct bucket through
// [plugin.Walk] and stamps the writer-shape metadata on each
// struct.
func (p *Plugin) Annotate(ctx *plugin.AnnotatorContext) error {
	return plugin.Walk(ctx, p)
}

// OnStruct is the [plugin.StructHook] entry point. It applies the
// directive overrides first (positive forces, negative suppresses)
// then falls through to the canonical signature heuristic.
func (*Plugin) OnStruct(_ *plugin.AnnotatorContext, s *node.Struct) {
	override, hasOverride := directiveOverride(s)
	if hasOverride && !override {
		Detected.Set(s.Meta(), false, Name)
		MethodQName.Set(s.Meta(), "", Name)
		return
	}
	method, ok := matchSignature(s)
	if hasOverride && override {
		// Positive override forces detection regardless of the
		// heuristic outcome; record the method when present so
		// downstream consumers can still navigate to a concrete
		// signature on directive-driven matches that happen to
		// also have one.
		Detected.Set(s.Meta(), true, Name)
		if ok {
			MethodQName.Set(s.Meta(), methodQName(s, method), Name)
		} else {
			MethodQName.Set(s.Meta(), "", Name)
		}
		return
	}
	if ok {
		Detected.Set(s.Meta(), true, Name)
		MethodQName.Set(s.Meta(), methodQName(s, method), Name)
		return
	}
	Detected.Set(s.Meta(), false, Name)
	MethodQName.Set(s.Meta(), "", Name)
}

// directiveOverride scans s for the writer directive. Returns
// (true, true) for `+gen:writer`, (false, true) for `-gen:writer`,
// and (false, false) when neither directive is attached.
func directiveOverride(s *node.Struct) (positive, hasOverride bool) {
	for _, d := range s.DirectiveList {
		if d.Name != DirectiveName {
			continue
		}
		return !d.Negated, true
	}
	return false, false
}

// matchSignature returns the method that matches the canonical
// io.Writer signature, or nil + false when none does. The match
// is exact: parameter type `[]byte`, return types `(int, error)`,
// method name `Write`.
func matchSignature(s *node.Struct) (*node.Method, bool) {
	for _, m := range s.Methods {
		if m.Name != writeMethodName || len(m.Params) != 1 || len(m.Returns) != 2 {
			continue
		}
		if !isByteSlice(m.Params[0].Type) {
			continue
		}
		if !isBuiltin(m.Returns[0], "int") || !isBuiltin(m.Returns[1], "error") {
			continue
		}
		return m, true
	}
	return nil, false
}

// isByteSlice reports whether ref is the unqualified `[]byte`
// type — a Slice variant carrying an Elem of `byte` (or its alias
// `uint8`).
func isByteSlice(ref *node.TypeRef) bool {
	if ref == nil || !ref.IsSlice() || ref.Elem == nil {
		return false
	}
	return isBuiltin(ref.Elem, "byte") || isBuiltin(ref.Elem, "uint8")
}

// isBuiltin reports whether ref is the unqualified builtin named
// want — Named variant, no package, exactly the given Name.
func isBuiltin(ref *node.TypeRef, want string) bool {
	if ref == nil || !ref.IsBuiltin() {
		return false
	}
	return ref.Name == want
}

// methodQName recomposes the store's canonical method-bucket key
// (`<ownerQName>.<methodName>`) for the matched method. Mirrors
// the format [store.NodeView.addMethod] uses.
func methodQName(s *node.Struct, m *node.Method) string {
	return fmt.Sprintf("%s.%s", s.QName(), m.Name)
}
