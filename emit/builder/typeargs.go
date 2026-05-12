// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// TypeArgsFromNodeParams projects a [node.TypeParam] slice into the
// parallel `[]emit.Ref` value suitable for spreading as the
// variadic typeArgs to [emit.External] / [emit.Internal] when
// instantiating a generic host with the source's own type
// parameters. Each parameter contributes a bare-name
// [emit.Builtin]; constraints are intentionally not threaded —
// emit-side instantiation uses positional arguments, not the
// declaration-side bound list.
//
// An empty input slice returns nil so callers can spread the
// result through variadic positions without conditional plumbing.
func TypeArgsFromNodeParams(params []*node.TypeParam) []emit.Ref {
	if len(params) == 0 {
		return nil
	}
	out := make([]emit.Ref, 0, len(params))
	for _, tp := range params {
		out = append(out, emit.Builtin(tp.Name))
	}
	return out
}

// TypeArgsFromEmitParams is the [emit.TypeParam] counterpart of
// [TypeArgsFromNodeParams]. Plugins consuming upstream-generated
// generic interfaces (the emit-side mock-target path) use this
// form because their input refs already live on the emit layer.
func TypeArgsFromEmitParams(params []*emit.TypeParam) []emit.Ref {
	if len(params) == 0 {
		return nil
	}
	out := make([]emit.Ref, 0, len(params))
	for _, tp := range params {
		out = append(out, emit.Builtin(tp.Name))
	}
	return out
}

// ApplyTypeArgs adapts ref to carry typeArgs at its trailing
// generic-instantiation slot. For an [emit.ExternalRef] the
// function returns a fresh External constructed with the supplied
// type-arg list, preserving Package and Name. For any other Ref
// variant the input passes through unchanged — callers that need
// type-arg threading on other ref shapes (TypeRef instances built
// via [emit.Internal], composite refs) should pass typeArgs at
// construction time instead.
//
// An empty typeArgs slice returns ref unchanged.
func ApplyTypeArgs(ref emit.Ref, typeArgs []emit.Ref) emit.Ref {
	if len(typeArgs) == 0 {
		return ref
	}
	if e, ok := ref.(*emit.ExternalRef); ok {
		return emit.External(e.Package, e.Name, typeArgs...)
	}
	return ref
}
