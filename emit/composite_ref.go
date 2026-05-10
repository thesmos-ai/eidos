// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// CompositeShape discriminates the variant a [CompositeRef] takes.
// The set covers the common composite type shapes across Go, Rust,
// and similar systems languages.
type CompositeShape int

// CompositeShape variants in declaration order.
const (
	// ShapePointer is a pointer to another ref (Go's *T).
	ShapePointer CompositeShape = iota
	// ShapeSlice is a variable-length sequence (Go's []T).
	ShapeSlice
	// ShapeArray is a fixed-length sequence (Go's [N]T). Length is
	// stored in [CompositeRef.ArrayLen].
	ShapeArray
	// ShapeMap is an associative container (Go's map[K]V).
	// MapKey / MapValue hold the key and value refs.
	ShapeMap
	// ShapeFunc is a function type. FuncParams and FuncReturns hold
	// the parameter and return refs.
	ShapeFunc
	// ShapeChan is a channel type (Go's chan T).
	ShapeChan
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
	case ShapeChan:
		return "chan"
	default:
		return "composite_shape(?)"
	}
}

// CompositeRef wraps an inner [Ref] with a composite shape. The
// [CompositeShape] discriminator selects which fields are
// meaningful — Pointer / Slice / Chan use Elem alone; Array uses
// Elem + ArrayLen; Map uses MapKey + MapValue; Func uses FuncParams
// + FuncReturns.
type CompositeRef struct {
	BaseEmit

	// Shape discriminates the composite variant.
	Shape CompositeShape

	// Elem is the element ref for Pointer / Slice / Array / Chan
	// shapes.
	Elem Ref

	// ArrayLen is the length of a fixed-size array. Meaningful only
	// when Shape == ShapeArray.
	ArrayLen int

	// MapKey and MapValue describe the Map shape's key and value
	// types.
	MapKey   Ref
	MapValue Ref

	// FuncParams and FuncReturns describe the Func shape's
	// parameter and return types in source order.
	FuncParams  []Ref
	FuncReturns []Ref
}

// Kind returns [KindCompositeRef].
func (*CompositeRef) Kind() directive.Kind { return KindCompositeRef }

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

// ChanOf wraps elem in a channel composite.
//
//	emit.ChanOf(emit.External("context", "Context")) // chan context.Context
func ChanOf(elem Ref) *CompositeRef {
	return &CompositeRef{Shape: ShapeChan, Elem: elem}
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
