// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
)

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
		if !golang.IsByteSlice(storefixture.Slice(&node.TypeRef{Name: "byte"})) {
			t.Errorf("[]byte must be recognised as byte slice")
		}
	})

	t.Run("string slice does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsByteSlice(storefixture.Slice(&node.TypeRef{Name: "string"})) {
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
		if !golang.IsSlice(storefixture.Slice(&node.TypeRef{Name: "string"})) {
			t.Errorf("[]string must be recognised as a (non-byte) slice")
		}
	})

	t.Run("byte slice does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsSlice(storefixture.Slice(&node.TypeRef{Name: "byte"})) {
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
		ref := storefixture.Map(&node.TypeRef{Name: "string"}, &node.TypeRef{Name: "int"})
		if !golang.IsMap(ref) {
			t.Errorf("map must be recognised")
		}
	})

	t.Run("slice does not match", func(t *testing.T) {
		t.Parallel()
		if golang.IsMap(storefixture.Slice(&node.TypeRef{Name: "string"})) {
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
	s := storefixture.New().
		Package("blog", "example.com/blog").
		Struct("Article", func(sb *storefixture.StructBuilder) {
			sb.Field("Title", &node.TypeRef{Name: "string"}, nil)
			sb.Field("internal", &node.TypeRef{Name: "string"}, nil)
			sb.Field("Body", &node.TypeRef{Name: "string"}, nil)
		}).
		PackageNode().Structs[0]

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
		s := storefixture.New().
			Package("blog", "example.com/blog").
			Struct("Article", func(sb *storefixture.StructBuilder) {}).
			PackageNode().Structs[0]
		if got := golang.TypeArgs(s); got != "" {
			t.Errorf("TypeArgs(non-generic) = %q, want empty", got)
		}
	})

	t.Run("single type parameter yields [T]", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("blog", "example.com/blog").
			Struct("Container", func(sb *storefixture.StructBuilder) {
				sb.TypeParam("T", nil)
			}).
			PackageNode().Structs[0]
		if got := golang.TypeArgs(s); got != "[T]" {
			t.Errorf("TypeArgs = %q, want [T]", got)
		}
	})

	t.Run("two type parameters yield [T, K]", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("blog", "example.com/blog").
			Struct("Map", func(sb *storefixture.StructBuilder) {
				sb.TypeParam("T", nil)
				sb.TypeParam("K", nil)
			}).
			PackageNode().Structs[0]
		if got := golang.TypeArgs(s); got != "[T, K]" {
			t.Errorf("TypeArgs = %q, want [T, K]", got)
		}
	})
}
