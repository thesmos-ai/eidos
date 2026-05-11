// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
)

// TestDocsAndDirectives covers the preferred-doc / spec+block
// directive walk.
func TestDocsAndDirectives(t *testing.T) {
	t.Parallel()
	t.Run("spec doc takes precedence over the block doc", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\n// block doc.\ntype (\n\t// spec doc.\n\tS struct{}\n)\n",
		})
		s := pkg.StructByName("S")
		if s == nil {
			t.Fatalf("S missing")
		}
		want := []string{"spec doc."}
		if !slices.Equal(s.DocLines, want) {
			t.Fatalf("DocLines = %v, want %v", s.DocLines, want)
		}
	})

	t.Run("block doc is used when the spec has no doc", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\n// block doc.\ntype (\n\tS struct{}\n)\n",
		})
		s := pkg.StructByName("S")
		want := []string{"block doc."}
		if !slices.Equal(s.DocLines, want) {
			t.Fatalf("DocLines = %v, want %v", s.DocLines, want)
		}
	})

	t.Run("directives parse off non-directive comments cleanly", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\n// +gen:mock\ntype I interface{ M() }\n",
		})
		i := pkg.InterfaceByName("I")
		if i == nil {
			t.Fatalf("I missing")
		}
		if len(i.DirectiveList) != 1 {
			t.Fatalf("expected 1 directive, got %d", len(i.DirectiveList))
		}
		if i.DirectiveList[0].Name != directive.Name("mock") {
			t.Fatalf("directive name = %q, want %q", i.DirectiveList[0].Name, "mock")
		}
	})
}

// TestParseDirectives covers the directive comment walker on inputs
// with several directives, one valid + one malformed, and a comment
// that simply isn't a directive.
func TestParseDirectives(t *testing.T) {
	t.Parallel()
	t.Run("non-directive comments produce no diagnostics", func(t *testing.T) {
		t.Parallel()
		s, d := loadFromSource(t, map[string]string{
			"a.go": "package a\n\n// A plain doc-comment line.\ntype S struct{}\n",
		})
		_ = s
		for _, dg := range d.Diagnostics() {
			t.Errorf("unexpected diagnostic: %v %v %v", dg.Severity, dg.Pos, dg.Message)
		}
	})

	t.Run("malformed directive emits a positioned diagnostic", func(t *testing.T) {
		t.Parallel()
		_, d := loadFromSource(t, map[string]string{
			"a.go": "package a\n\n// +gen:\ntype S struct{}\n",
		})
		if !d.HasErrors() {
			t.Fatalf("expected an Error diagnostic for empty directive name")
		}
	})

	t.Run("multiple valid directives are recorded in order", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\n// +gen:mock\n// +gen:repo\ntype I interface{ M() }\n",
		})
		i := pkg.InterfaceByName("I")
		names := []directive.Name{i.DirectiveList[0].Name, i.DirectiveList[1].Name}
		want := []directive.Name{"mock", "repo"}
		if !slices.Equal(names, want) {
			t.Fatalf("directives = %v, want %v", names, want)
		}
	})
}

// TestPreferred verifies the [preferred] helper picks the first
// non-empty comment group from its alternatives.
func TestPreferred(t *testing.T) {
	t.Parallel()
	t.Run("missing both yields nil docs", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype S struct{}\n",
		})
		if pkg.StructByName("S").DocLines != nil {
			t.Fatalf("expected nil DocLines on undocumented struct")
		}
	})
}
