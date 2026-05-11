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

// TypeParamRef returns a use-site reference to a generic type
// parameter ([node.TypeRefTypeParam]). The Name carries the
// declaring TypeParam's identifier — e.g. `TypeParamRef("T")` for
// the receiver-side `T` in `func (l *List[T]) Get() T`.
func TypeParamRef(name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefTypeParam, Name: name}
}

// AnonStruct returns an inline anonymous-struct type reference
// ([node.TypeRefAnonStruct]) carrying the supplied fields and embeds.
// Each Field's Owner back-pointer is wired to the returned ref so
// consumers walking up from a field locate the anonymous host.
func AnonStruct(fields []*node.Field, embeds []*node.Embed) *node.TypeRef {
	r := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Fields: fields, Embeds: embeds}
	for _, f := range fields {
		f.Owner = r
	}
	for _, e := range embeds {
		e.Owner = r
	}
	return r
}

// AnonInterface returns an inline anonymous-interface type reference
// ([node.TypeRefAnonInterface]) carrying the supplied methods and
// embeds. Each Method's and Embed's Owner back-pointer is wired to
// the returned ref.
func AnonInterface(methods []*node.Method, embeds []*node.Embed) *node.TypeRef {
	r := &node.TypeRef{TypeKind: node.TypeRefAnonInterface, Methods: methods, Embeds: embeds}
	for _, m := range methods {
		m.Owner = r
	}
	for _, e := range embeds {
		e.Owner = r
	}
	return r
}

// Constraint returns a [node.Constraint] with the supplied refs as
// embedded named bounds — the universal shape across languages
// (interfaces, traits, protocols, the `comparable` predeclared
// identifier). A nil return value reads as "any" via
// [node.Constraint.IsAny].
func Constraint(embeds ...*node.TypeRef) *node.Constraint {
	return &node.Constraint{Embedded: embeds}
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

// Func returns a function-type [node.TypeRef] with the supplied
// parameter and return types.
func Func(params, returns []*node.TypeRef) *node.TypeRef {
	return &node.TypeRef{
		TypeKind:    node.TypeRefFunc,
		FuncParams:  params,
		FuncReturns: returns,
	}
}
