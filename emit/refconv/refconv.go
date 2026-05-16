// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package refconv lifts language-agnostic [node.TypeRef] values into
// concrete [emit.Ref] values. Plugins that consume source-side
// declarations share the conversion through this package so the
// rules stay aligned across every consumer — a new TypeRef variant
// lands here once and every plugin inherits the support.
//
// The produced emit ref's OriginNode points back at the source
// [node.TypeRef] so backends and downstream consumers can reach
// source-side meta (e.g., bridge-annotator-stamped `go.type` for
// proto→Go translation).
package refconv

import (
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// FromNode returns the [emit.Ref] equivalent of r. Builtin and named
// refs map to [emit.Builtin] and [emit.External]; composites map to
// their corresponding emit constructors. Type parameters render as
// unqualified identifiers — sufficient for the in-method context
// most plugins emit. The frontend guarantees non-nil refs for
// every parsed type, so the function does not guard against a nil
// receiver.
//
// The produced emit ref's OriginNode points back at r so backends
// and downstream consumers can reach the source-side meta.
func FromNode(r *node.TypeRef) emit.Ref {
	ref := liftFromNode(r)
	setOrigin(ref, r)
	return ref
}

// liftFromNode performs the variant-by-variant conversion. Wrapped
// in [FromNode] so the OriginNode threading lands at a single
// site rather than at each constructor call.
func liftFromNode(r *node.TypeRef) emit.Ref {
	switch {
	case r.IsPointer():
		return emit.Ptr(FromNode(r.Elem))
	case r.IsSlice():
		return emit.SliceOf(FromNode(r.Elem))
	case r.IsArray():
		return emit.ArrayOf(FromNode(r.Elem), r.ArrayLen)
	case r.IsMap():
		return emit.MapOf(FromNode(r.MapKey), FromNode(r.MapValue))
	case r.IsBuiltin():
		return emit.Builtin(r.Name)
	case r.IsTypeParam():
		return emit.Builtin(r.Name)
	case r.IsAnonInterface():
		// Anonymous interfaces sit in two practical buckets at the
		// plugin tier: the empty interface (the constraint-fallback
		// shape the Go frontend emits for the predeclared `any`)
		// and inline interfaces with methods or embeds (which
		// plugins do not yet preserve structurally). For the
		// former, render through the `any` builtin so
		// type-parameter constraints round-trip correctly; for
		// the latter, fall back to the same keyword — the
		// rendered output keeps compiling at the cost of losing
		// the inline shape, which is a separate follow-up.
		return emit.Builtin("any")
	}
	args := make([]emit.Ref, 0, len(r.TypeArgs))
	for _, a := range r.TypeArgs {
		args = append(args, FromNode(a))
	}
	return emit.External(r.Package, r.Name, args...)
}

// setOrigin records r as the OriginNode of ref on every concrete
// [emit.Ref] implementation. The switch enumerates the closed set
// of types [liftFromNode] can produce; future emit variants need
// a matching arm here so source-side meta stays reachable from
// the produced emit graph.
func setOrigin(ref emit.Ref, r *node.TypeRef) {
	switch v := ref.(type) {
	case *emit.BuiltinRef:
		v.OriginNode = r
	case *emit.ExternalRef:
		v.OriginNode = r
	case *emit.CompositeRef:
		v.OriginNode = r
	case *emit.TypeRef:
		v.OriginNode = r
	}
}

// ConstraintFromNode lifts a [node.Constraint] into its [emit.Constraint]
// equivalent. The any-constraint shape (nil receiver or no Embedded
// entries) lifts to nil — callers passing the result into a builder
// setter that documents "nil means any" get the right behaviour
// without further branching.
func ConstraintFromNode(c *node.Constraint) *emit.Constraint {
	if c == nil || c.IsAny() {
		return nil
	}
	refs := make([]emit.Ref, 0, len(c.Embedded))
	for _, e := range c.Embedded {
		refs = append(refs, FromNode(e))
	}
	return &emit.Constraint{Raw: c.Raw, Embedded: refs}
}
