// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/kind"
)

// CompositeShape discriminates the variant a [CompositeRef] takes.
// The set covers composite type shapes common to general-purpose
// languages — Go, Rust, TypeScript, and similar. Language-specific
// composites (Go-style channels, Rust enum variants with payloads,
// etc.) ride on plugin-defined emit kinds rather than first-class
// variants here.
type CompositeShape int

// CompositeShape variants in declaration order.
const (
	// ShapePointer is a pointer to another ref.
	ShapePointer CompositeShape = iota
	// ShapeSlice is a variable-length sequence.
	ShapeSlice
	// ShapeArray is a fixed-length sequence. Length is stored in
	// [CompositeRef.ArrayLen].
	ShapeArray
	// ShapeMap is an associative container. MapKey / MapValue hold
	// the key and value refs.
	ShapeMap
	// ShapeFunc is a function type. FuncParams and FuncReturns hold
	// the parameter and return refs.
	ShapeFunc
	// ShapeUnion is a union of terms, used for constraint interfaces
	// of the form `A | B | ~C`. UnionTerms holds one [UnionTerm] per
	// member of the union; the per-term Approx flag records whether
	// the source carried the `~` prefix.
	ShapeUnion
)

// String returns the lower-case textual form of s for diagnostics.
func (s CompositeShape) String() string {
	switch s {
	case ShapePointer:
		return "pointer"
	case ShapeSlice:
		return "slice"
	case ShapeArray:
		return "array"
	case ShapeMap:
		return "map"
	case ShapeFunc:
		return "func"
	case ShapeUnion:
		return "union"
	default:
		return "composite_shape(?)"
	}
}

// UnionTerm is one member of a [ShapeUnion] composite. Approx records
// whether the source term carried the Go `~` prefix (e.g., `~int`)
// — meaningful when the union appears as a constraint interface in
// a type parameter list.
type UnionTerm struct {
	Type   Ref  `json:"-"`
	Approx bool `json:"approx,omitempty"`
}

// CompositeRef wraps an inner [Ref] with a composite shape. The
// [CompositeShape] discriminator selects which fields are
// meaningful — Pointer / Slice use Elem; Array uses Elem +
// ArrayLen; Map uses MapKey + MapValue; Func uses FuncParams +
// FuncReturns; Union uses UnionTerms.
type CompositeRef struct {
	BaseEmit

	// Shape discriminates the composite variant.
	Shape CompositeShape `json:"shape"`

	// Elem is the element ref for Pointer / Slice / Array shapes.
	Elem Ref `json:"-"`

	// ArrayLen is the length of a fixed-size array. Meaningful only
	// when Shape == ShapeArray.
	ArrayLen int `json:"array_len,omitempty"`

	// MapKey and MapValue describe the Map shape's key and value
	// types.
	MapKey   Ref `json:"-"`
	MapValue Ref `json:"-"`

	// FuncParams and FuncReturns describe the Func shape's
	// parameter and return types in source order.
	FuncParams  []Ref `json:"-"`
	FuncReturns []Ref `json:"-"`

	// UnionTerms holds the members of a [ShapeUnion] composite in
	// source order. Each term carries its element ref plus an
	// Approx flag (the Go `~T` prefix).
	UnionTerms []UnionTerm `json:"union_terms,omitempty"`
}

// Kind returns [KindCompositeRef].
func (*CompositeRef) Kind() kind.Kind { return KindCompositeRef }

// isRef marks CompositeRef as a [Ref] implementation.
func (*CompositeRef) isRef() {}

// Ptr wraps elem in a pointer composite.
//
//	emit.Ptr(emit.External("database/sql", "DB")) // *sql.DB
func Ptr(elem Ref) *CompositeRef {
	return &CompositeRef{Shape: ShapePointer, Elem: elem}
}

// SliceOf wraps elem in a slice composite.
//
//	emit.SliceOf(emit.Builtin("string")) // []string
func SliceOf(elem Ref) *CompositeRef {
	return &CompositeRef{Shape: ShapeSlice, Elem: elem}
}

// ArrayOf wraps elem in a fixed-length array composite of the given
// length.
//
//	emit.ArrayOf(emit.Builtin("byte"), 16) // [16]byte
func ArrayOf(elem Ref, length int) *CompositeRef {
	return &CompositeRef{Shape: ShapeArray, Elem: elem, ArrayLen: length}
}

// MapOf builds a map composite with the given key and value refs.
//
//	emit.MapOf(emit.Builtin("string"), emit.Internal(userStruct))
func MapOf(key, value Ref) *CompositeRef {
	return &CompositeRef{Shape: ShapeMap, MapKey: key, MapValue: value}
}

// FuncOf builds a function composite with the given parameter and
// return refs in source order. Nil slices are normalised to empty
// slices.
//
//	emit.FuncOf(
//	    []emit.Ref{emit.External("context", "Context")},
//	    []emit.Ref{emit.Builtin("error")},
//	) // func(context.Context) error
func FuncOf(params, returns []Ref) *CompositeRef {
	if params == nil {
		params = []Ref{}
	}
	if returns == nil {
		returns = []Ref{}
	}
	return &CompositeRef{Shape: ShapeFunc, FuncParams: params, FuncReturns: returns}
}

// Union builds a union composite from the supplied terms in source
// order. Variadic zero-term invocation produces a non-nil empty
// UnionTerms slice — useful as a builder seed that callers append
// to before downstream consumption.
//
//	emit.Union(
//	    emit.UnionTerm{Type: emit.Builtin("int")},
//	    emit.UnionTerm{Type: emit.Builtin("string"), Approx: true},
//	) // int | ~string
func Union(terms ...UnionTerm) *CompositeRef {
	if terms == nil {
		terms = []UnionTerm{}
	}
	return &CompositeRef{Shape: ShapeUnion, UnionTerms: terms}
}
