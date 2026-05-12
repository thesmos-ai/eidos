// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package refconv lifts language-agnostic [node.TypeRef] values into
// concrete [emit.Ref] values. Reference plugins that consume source
// declarations (buildergen, mockgen) share the conversion through
// this package so the rules stay aligned across the reference
// surface — a new TypeRef variant lands here once and every consumer
// inherits the support.
package refconv

import (
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// FromNode returns the [emit.Ref] equivalent of r. Builtin and named
// refs map to [emit.Builtin] and [emit.External]; composites map to
// their corresponding emit constructors. Type parameters render as
// unqualified identifiers — sufficient for the in-method context
// reference plugins emit. The frontend guarantees non-nil refs for
// every parsed type, so the function does not guard against a nil
// receiver.
func FromNode(r *node.TypeRef) emit.Ref {
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
	}
	args := make([]emit.Ref, 0, len(r.TypeArgs))
	for _, a := range r.TypeArgs {
		args = append(args, FromNode(a))
	}
	return emit.External(r.Package, r.Name, args...)
}
