// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture

import "go.thesmos.sh/eidos/node"

// Named returns a Named [node.TypeRef] with no package — the
// frontend-conventional shape for primitive and in-scope types
// ("int", "string", "any", in-package "User").
func Named(name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Name: name}
}

// PkgNamed returns a Named [node.TypeRef] qualified by a package
// path. Use this for cross-package references such as
// `context.Context` or `time.Time`.
func PkgNamed(pkg, name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Package: pkg, Name: name}
}

// WithArgs returns a copy of named with the supplied generic type
// arguments. named must be a Named ref; calling WithArgs on a
// non-Named ref panics — the shape is invalid for any other variant.
func WithArgs(named *node.TypeRef, args ...*node.TypeRef) *node.TypeRef {
	if named == nil || named.TypeKind != node.TypeRefNamed {
		// Test-only fixture; callers expect a panic on misuse.
		panic("storefixture: WithArgs requires a Named TypeRef") //nolint:forbidigo
	}
	clone := *named
	clone.TypeArgs = append([]*node.TypeRef(nil), args...)
	return &clone
}

// Pointer returns a pointer [node.TypeRef] over elem.
func Pointer(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: elem}
}

// Slice returns a slice [node.TypeRef] over elem.
func Slice(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: elem}
}

// Array returns a fixed-length array [node.TypeRef] of length n over
// elem.
func Array(elem *node.TypeRef, n int) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefArray, Elem: elem, ArrayLen: n}
}

// Map returns a map [node.TypeRef] keyed by key with value type
// value.
func Map(key, value *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefMap, MapKey: key, MapValue: value}
}

// Chan returns a channel [node.TypeRef] over elem. Channel
// directionality is not yet modelled at the type-ref level; tests
// that need it should attach metadata to the returned ref.
func Chan(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefChan, Elem: elem}
}

// Func returns a function-type [node.TypeRef] with the supplied
// parameter and return types.
func Func(params, returns []*node.TypeRef) *node.TypeRef {
	return &node.TypeRef{
		TypeKind:    node.TypeRefFunc,
		FuncParams:  params,
		FuncReturns: returns,
	}
}
