// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
)

// sliceRef returns a [node.TypeRef] of the given element type —
// the inline equivalent of storefixture.Slice. Kept local so
// the package stays a leaf with no cross-module test
// dependencies.
func sliceRef(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: elem}
}

// mapRef returns a [node.TypeRef] of the given key+value types
// — the inline equivalent of storefixture.Map.
func mapRef(k, v *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefMap, MapKey: k, MapValue: v}
}

// TestIsExported pins Go's exported-identifier rule. The
// upstream consumers (plugin templates, [ExportedFields])
// route every field-filter / setter-emission decision
// through this helper, so the contract has to be
// unambiguous: first ASCII upper-case rune is exported,
// everything else is not, empty string is not.
func TestIsExported(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		want bool
	}{
		{"Title", true},
		{"ID", true},
		{"internal", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := golang.IsExported(tc.name); got != tc.want {
				t.Errorf("IsExported(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestIsByteSlice pins the `[]byte` discrimination — the
// only slice shape Go renders idiomatically through a
// bytes-string convenience setter pair. Plugin templates
// branch on this helper to pick the setter shape per field.
func TestIsByteSlice(t *testing.T) {
	t.Parallel()

	t.Run("bytes slice matches", func(t *testing.T) {
		t.Parallel()
		if !golang.IsByteSlice(sliceRef(&node.TypeRef{Name: "byte"})) {
			t.Errorf("[]byte must be recognised as byte slice")
		}
	})

	t.Run("string slice does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsByteSlice(sliceRef(&node.TypeRef{Name: "string"})) {
			t.Errorf("[]string must not be recognised as byte slice")
		}
	})

	t.Run("nil ref does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsByteSlice(nil) {
			t.Errorf("nil ref must not be recognised as byte slice")
		}
	})

	t.Run("non-slice does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsByteSlice(&node.TypeRef{Name: "byte"}) {
			t.Errorf("non-slice byte ref must not be recognised as byte slice")
		}
	})
}

// TestIsSlice / TestIsMap pin the IsByteSlice complement and
// the map-type predicate respectively. The non-byte-slice
// constraint matters because the bytes branch has its own
// setter shape; emitting both would render duplicate
// methods.
func TestIsSlice(t *testing.T) {
	t.Parallel()

	t.Run("string slice matches", func(t *testing.T) {
		t.Parallel()
		if !golang.IsSlice(sliceRef(&node.TypeRef{Name: "string"})) {
			t.Errorf("[]string must be recognised as a (non-byte) slice")
		}
	})

	t.Run("byte slice does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsSlice(sliceRef(&node.TypeRef{Name: "byte"})) {
			t.Errorf("[]byte must route through IsByteSlice, not IsSlice")
		}
	})

	t.Run("scalar does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsSlice(&node.TypeRef{Name: "string"}) {
			t.Errorf("scalar must not be recognised as a slice")
		}
	})
}

// TestIsMap covers the map-type predicate paired with the
// map's per-entry setter rendering in plugin templates.
func TestIsMap(t *testing.T) {
	t.Parallel()

	t.Run("map matches", func(t *testing.T) {
		t.Parallel()
		ref := mapRef(&node.TypeRef{Name: "string"}, &node.TypeRef{Name: "int"})
		if !golang.IsMap(ref) {
			t.Errorf("map must be recognised")
		}
	})

	t.Run("slice does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsMap(sliceRef(&node.TypeRef{Name: "string"})) {
			t.Errorf("slice must not be recognised as a map")
		}
	})

	t.Run("nil ref does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsMap(nil) {
			t.Errorf("nil ref must not be recognised as a map")
		}
	})
}

// TestExportedFields pins the source-order filter
// [ExportedFields] applies to a struct's full field list.
// Unexported fields drop out; exported fields keep their
// declared order so generated setters mirror the source's
// shape.
func TestExportedFields(t *testing.T) {
	t.Parallel()
	s := &node.Struct{
		Name:    "Article",
		Package: "example.com/blog",
		Fields: []*node.Field{
			{Name: "Title", Type: &node.TypeRef{Name: "string"}},
			{Name: "internal", Type: &node.TypeRef{Name: "string"}},
			{Name: "Body", Type: &node.TypeRef{Name: "string"}},
		},
	}
	got := golang.ExportedFields(s)
	if len(got) != 2 {
		t.Fatalf("ExportedFields = %d entries, want 2", len(got))
	}
	if got[0].Name != "Title" || got[1].Name != "Body" {
		t.Errorf("ExportedFields order = [%s, %s], want [Title, Body]", got[0].Name, got[1].Name)
	}
}

// TestTypeArgs pins the bracketed parameter-name use form
// the per-language template appends to receiver / return /
// composite refs in generic-struct emissions. Non-generic
// structs return the empty string so non-generic templates
// stay free of stray brackets.
func TestTypeArgs(t *testing.T) {
	t.Parallel()

	t.Run("non-generic struct yields empty string", func(t *testing.T) {
		t.Parallel()
		s := &node.Struct{Name: "Article", Package: "example.com/blog"}
		if got := golang.TypeArgs(s); got != "" {
			t.Errorf("TypeArgs(non-generic) = %q, want empty", got)
		}
	})

	t.Run("single type parameter yields [T]", func(t *testing.T) {
		t.Parallel()
		s := &node.Struct{
			Name:       "Container",
			Package:    "example.com/blog",
			TypeParams: []*node.TypeParam{{Name: "T"}},
		}
		if got := golang.TypeArgs(s); got != "[T]" {
			t.Errorf("TypeArgs = %q, want [T]", got)
		}
	})

	t.Run("two type parameters yield [T, K]", func(t *testing.T) {
		t.Parallel()
		s := &node.Struct{
			Name:    "Map",
			Package: "example.com/blog",
			TypeParams: []*node.TypeParam{
				{Name: "T"},
				{Name: "K"},
			},
		}
		if got := golang.TypeArgs(s); got != "[T, K]" {
			t.Errorf("TypeArgs = %q, want [T, K]", got)
		}
	})
}
