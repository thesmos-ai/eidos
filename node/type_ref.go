// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// TypeRefKind discriminates the variant forms a [TypeRef] can take.
// The set covers the type shapes common to Go, Rust, TypeScript, and
// similar systems languages; truly language-specific quirks ride on
// the TypeRef's metadata rather than being elevated to first-class
// fields.
type TypeRefKind int

// TypeRef variants in declaration order. Add new variants at the end
// so existing ordinals stay stable.
const (
	// TypeRefNamed is a reference by qualified name (Package.Name),
	// optionally with generic type arguments. Covers all
	// user-defined types and basic types ("int", "string", "any").
	TypeRefNamed TypeRefKind = iota
	// TypeRefPointer is a pointer to a referenced type (Go's *T,
	// Rust's &T / &mut T modelled via metadata).
	TypeRefPointer
	// TypeRefSlice is an ordered, variable-length sequence
	// (Go's []T, Rust's Vec<T> / [T], TypeScript's Array<T>).
	TypeRefSlice
	// TypeRefArray is a fixed-length sequence (Go's [N]T,
	// Rust's [T; N]).
	TypeRefArray
	// TypeRefMap is an associative container (Go's map[K]V,
	// Rust's HashMap<K, V>).
	TypeRefMap
	// TypeRefFunc is a function type with parameter and return
	// types.
	TypeRefFunc
	// TypeRefChan is a channel type (Go-specific; other backends
	// may reject these or map them via plugins).
	TypeRefChan
)

// String returns the lower-case textual form of k for diagnostics.
func (k TypeRefKind) String() string {
	switch k {
	case TypeRefNamed:
		return "named"
	case TypeRefPointer:
		return "pointer"
	case TypeRefSlice:
		return "slice"
	case TypeRefArray:
		return "array"
	case TypeRefMap:
		return "map"
	case TypeRefFunc:
		return "func"
	case TypeRefChan:
		return "chan"
	default:
		return "type_ref_kind(?)"
	}
}

// TypeRef is a unified type reference used by [Field], [Method],
// [Param], [Function], [Alias], and others. Its [TypeRefKind]
// discriminator determines which fields are meaningful.
//
// Most fields are nil / zero unless the kind uses them. The frontend
// populates only the relevant ones for each variant.
type TypeRef struct {
	BaseNode

	// TypeKind discriminates the variant. Set by the constructing
	// frontend; downstream reads via TypeRef.Kind().
	TypeKind TypeRefKind

	// Package is the source package path for Named refs ("" for
	// builtin or in-package types). Empty for non-Named variants.
	Package string

	// Name is the type identifier for Named refs. Empty for
	// non-Named variants.
	Name string

	// TypeArgs are generic type arguments for Named refs that are
	// generic instantiations (Go's `Map[string, int]`).
	TypeArgs []*TypeRef

	// Elem is the element type for Pointer / Slice / Array / Chan.
	Elem *TypeRef

	// ArrayLen is the length of an Array. Zero for slices and
	// non-array variants.
	ArrayLen int

	// MapKey is the key type for Map variants.
	MapKey *TypeRef

	// MapValue is the value type for Map variants.
	MapValue *TypeRef

	// FuncParams are the parameter types of a Func variant in
	// source order.
	FuncParams []*TypeRef

	// FuncReturns are the return types of a Func variant in source
	// order.
	FuncReturns []*TypeRef
}

// Kind returns [KindTypeRef] regardless of TypeRefKind — TypeRefKind
// discriminates *within* the type-reference family, not across the
// node hierarchy.
func (*TypeRef) Kind() directive.Kind { return KindTypeRef }

// IsBuiltin reports whether the ref is a Named ref with no Package
// (covering "int", "string", "bool", "any", and similar basic types).
func (r *TypeRef) IsBuiltin() bool {
	return r.TypeKind == TypeRefNamed && r.Package == ""
}

// IsGeneric reports whether the ref is a Named ref carrying generic
// type arguments.
func (r *TypeRef) IsGeneric() bool {
	return r.TypeKind == TypeRefNamed && len(r.TypeArgs) > 0
}

// IsPointer reports whether the ref is a pointer to another type.
func (r *TypeRef) IsPointer() bool { return r.TypeKind == TypeRefPointer }

// IsSlice reports whether the ref is a variable-length sequence.
func (r *TypeRef) IsSlice() bool { return r.TypeKind == TypeRefSlice }

// IsArray reports whether the ref is a fixed-length sequence.
func (r *TypeRef) IsArray() bool { return r.TypeKind == TypeRefArray }

// IsMap reports whether the ref is an associative container.
func (r *TypeRef) IsMap() bool { return r.TypeKind == TypeRefMap }

// IsFunc reports whether the ref is a function type.
func (r *TypeRef) IsFunc() bool { return r.TypeKind == TypeRefFunc }

// IsChan reports whether the ref is a channel type.
func (r *TypeRef) IsChan() bool { return r.TypeKind == TypeRefChan }

// Equal reports whether r and other refer to the same type
// structurally — same kind, same package + name + type args (for
// Named refs), same recursive shape (for composite refs).
//
// Equal compares structural type identity only; it does not consider
// position, doc comments, directives, or metadata. Two refs that
// describe the same logical type but originated from different
// positions in source compare equal — appropriate for "do these two
// methods have the same signature?" checks.
//
// nil compares equal to nil; a nil receiver or nil argument
// compares unequal to any non-nil ref.
func (r *TypeRef) Equal(other *TypeRef) bool {
	if r == nil || other == nil {
		return r == other
	}
	if r.TypeKind != other.TypeKind {
		return false
	}
	switch r.TypeKind {
	case TypeRefNamed:
		return r.equalNamed(other)
	case TypeRefPointer, TypeRefSlice, TypeRefChan:
		return r.Elem.Equal(other.Elem)
	case TypeRefArray:
		return r.ArrayLen == other.ArrayLen && r.Elem.Equal(other.Elem)
	case TypeRefMap:
		return r.MapKey.Equal(other.MapKey) && r.MapValue.Equal(other.MapValue)
	case TypeRefFunc:
		return equalRefSlice(r.FuncParams, other.FuncParams) &&
			equalRefSlice(r.FuncReturns, other.FuncReturns)
	default:
		return false
	}
}

// equalNamed handles the [TypeRefNamed] equality path: package, name,
// and recursive comparison of type arguments.
func (r *TypeRef) equalNamed(other *TypeRef) bool {
	if r.Package != other.Package || r.Name != other.Name {
		return false
	}
	return equalRefSlice(r.TypeArgs, other.TypeArgs)
}

// equalRefSlice compares two TypeRef slices element-wise for
// structural equality.
func equalRefSlice(a, b []*TypeRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}
