// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestCompositeShape_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    emit.CompositeShape
		want string
	}{
		{"Pointer", emit.ShapePointer, "pointer"},
		{"Slice", emit.ShapeSlice, "slice"},
		{"Array", emit.ShapeArray, "array"},
		{"Map", emit.ShapeMap, "map"},
		{"Func", emit.ShapeFunc, "func"},
		{"Union", emit.ShapeUnion, "union"},
		{"unknown stringifies with a marker", emit.CompositeShape(99), "composite_shape(?)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.s.String(), tc.want)
		})
	}
}

func TestPtr(t *testing.T) {
	t.Parallel()

	t.Run("wraps elem in a pointer composite", func(t *testing.T) {
		t.Parallel()
		r := emit.Ptr(emit.Builtin("int"))
		if r.Shape != emit.ShapePointer {
			t.Fatalf("Shape = %s, want pointer", r.Shape)
		}
		if r.Elem == nil {
			t.Fatalf("Elem should be the supplied ref")
		}
	})
}

func TestSliceOf(t *testing.T) {
	t.Parallel()

	t.Run("wraps elem in a slice composite", func(t *testing.T) {
		t.Parallel()
		r := emit.SliceOf(emit.Builtin("byte"))
		if r.Shape != emit.ShapeSlice {
			t.Fatalf("Shape = %s, want slice", r.Shape)
		}
	})
}

func TestArrayOf(t *testing.T) {
	t.Parallel()

	t.Run("wraps elem in a fixed-length array composite", func(t *testing.T) {
		t.Parallel()
		r := emit.ArrayOf(emit.Builtin("byte"), 16)
		if r.Shape != emit.ShapeArray {
			t.Fatalf("Shape = %s, want array", r.Shape)
		}
		if r.ArrayLen != 16 {
			t.Fatalf("ArrayLen = %d, want 16", r.ArrayLen)
		}
	})
}

func TestMapOf(t *testing.T) {
	t.Parallel()

	t.Run("wraps key and value refs in a map composite", func(t *testing.T) {
		t.Parallel()
		r := emit.MapOf(emit.Builtin("string"), emit.Builtin("int"))
		if r.Shape != emit.ShapeMap {
			t.Fatalf("Shape = %s, want map", r.Shape)
		}
		if r.MapKey == nil || r.MapValue == nil {
			t.Fatalf("MapKey/MapValue must be populated")
		}
	})
}

func TestFuncOf(t *testing.T) {
	t.Parallel()

	t.Run("constructs a function composite from params and returns", func(t *testing.T) {
		t.Parallel()
		r := emit.FuncOf(
			[]emit.Ref{emit.Builtin("int")},
			[]emit.Ref{emit.Builtin("error")},
		)
		if r.Shape != emit.ShapeFunc {
			t.Fatalf("Shape = %s, want func", r.Shape)
		}
		if len(r.FuncParams) != 1 || len(r.FuncReturns) != 1 {
			t.Fatalf("FuncParams/Returns mismatch: %+v", r)
		}
	})

	t.Run("normalises nil slices to empty slices", func(t *testing.T) {
		t.Parallel()
		r := emit.FuncOf(nil, nil)
		if r.FuncParams == nil || r.FuncReturns == nil {
			t.Fatalf("nil slices should normalise to empty, not nil")
		}
		if len(r.FuncParams) != 0 || len(r.FuncReturns) != 0 {
			t.Fatalf("expected empty slices; got %+v", r)
		}
	})
}

func TestUnion(t *testing.T) {
	t.Parallel()

	t.Run("constructs a union composite with terms and approx flags", func(t *testing.T) {
		t.Parallel()
		r := emit.Union(
			emit.UnionTerm{Type: emit.Builtin("int")},
			emit.UnionTerm{Type: emit.Builtin("string"), Approx: true},
		)
		if r.Shape != emit.ShapeUnion {
			t.Fatalf("Shape = %s, want union", r.Shape)
		}
		if len(r.UnionTerms) != 2 {
			t.Fatalf("UnionTerms len = %d, want 2", len(r.UnionTerms))
		}
		if r.UnionTerms[0].Approx {
			t.Fatalf("first term should not be approx")
		}
		if !r.UnionTerms[1].Approx {
			t.Fatalf("second term should be approx")
		}
		if r.UnionTerms[0].Type == nil || r.UnionTerms[1].Type == nil {
			t.Fatalf("term Type refs must be populated")
		}
	})

	t.Run("zero terms produce a non-nil empty slice", func(t *testing.T) {
		t.Parallel()
		r := emit.Union()
		if r.Shape != emit.ShapeUnion {
			t.Fatalf("Shape = %s, want union", r.Shape)
		}
		if r.UnionTerms == nil {
			t.Fatalf("UnionTerms should not be nil even with zero terms")
		}
		if len(r.UnionTerms) != 0 {
			t.Fatalf("expected empty UnionTerms; got %+v", r.UnionTerms)
		}
	})

	t.Run("reports KindCompositeRef", func(t *testing.T) {
		t.Parallel()
		r := emit.Union(emit.UnionTerm{Type: emit.Builtin("int")})
		if r.Kind() != emit.KindCompositeRef {
			t.Fatalf("Kind = %s, want %s", r.Kind(), emit.KindCompositeRef)
		}
	})

	t.Run("satisfies the Ref interface", func(t *testing.T) {
		t.Parallel()
		var _ emit.Ref = emit.Union(emit.UnionTerm{Type: emit.Builtin("int")})
	})
}

func TestCompositeRef_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindCompositeRef", func(t *testing.T) {
		t.Parallel()
		r := emit.Ptr(emit.Builtin("int"))
		if r.Kind() != emit.KindCompositeRef {
			t.Fatalf("Kind = %s, want %s", r.Kind(), emit.KindCompositeRef)
		}
	})
}

func TestCompositeRef_SatisfiesRef(t *testing.T) {
	t.Parallel()

	t.Run("CompositeRef satisfies the Ref interface", func(t *testing.T) {
		t.Parallel()
		var _ emit.Ref = emit.Ptr(emit.Builtin("int"))
	})
}
