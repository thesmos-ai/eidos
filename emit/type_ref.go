// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// TypeRef references another emit entity in the same generation run
// — most commonly a [Struct], [Interface], [Alias], or [Enum] that
// the same (or an earlier) generator produced. Backends resolve
// these without emitting any import statement: the target's name is
// already in scope of the generated file.
//
// TypeArgs holds generic instantiation arguments
// (Container[int, string] → TypeArgs of [BuiltinRef("int"),
// BuiltinRef("string")]). Empty for non-generic references.
type TypeRef struct {
	BaseEmit
	Target   Node
	TypeArgs []Ref
}

// Kind returns [KindTypeRef].
func (*TypeRef) Kind() directive.Kind { return KindTypeRef }

// isRef marks TypeRef as a [Ref] implementation.
func (*TypeRef) isRef() {}

// Internal constructs an internal TypeRef pointing at target. Pass
// optional TypeArgs for generic instantiation. The returned value is
// a fresh allocation suitable for immediate use in emit construction.
//
//	repoRef := emit.Internal(userRepoStruct)
//	mapEntry := emit.Internal(entryStruct, emit.Builtin("string"))
func Internal(target Node, typeArgs ...Ref) *TypeRef {
	return &TypeRef{Target: target, TypeArgs: typeArgs}
}
