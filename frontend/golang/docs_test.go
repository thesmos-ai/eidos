// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"slices"
	"testing"
)

// TestDocLinesFromCommentGroup exercises every documented comment
// form via the public Load surface — `//` line comments, multi-line
// `/* … */` block comments, and the leading-space stripping
// convention.
func TestDocLinesFromCommentGroup(t *testing.T) {
	t.Parallel()
	t.Run("line comments preserve content one entry per line", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\n// First line.\n// Second line.\ntype S struct{}\n",
		})
		s := pkg.StructByName("S")
		if s == nil {
			t.Fatalf("S missing")
		}
		want := []string{"First line.", "Second line."}
		if !slices.Equal(s.DocLines, want) {
			t.Fatalf("DocLines = %v, want %v", s.DocLines, want)
		}
	})

	t.Run("block comments split on newlines and strip * continuation", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\n/*\nFirst.\n* Second.\n*/\ntype S struct{}\n",
		})
		s := pkg.StructByName("S")
		if s == nil {
			t.Fatalf("S missing")
		}
		// The block-body splitter trims the closing-marker blank
		// line and the continuation `* ` prefix.
		want := []string{"", "First.", "Second."}
		if !slices.Equal(s.DocLines, want) {
			t.Fatalf("DocLines = %v, want %v", s.DocLines, want)
		}
	})

	t.Run("absent doc yields nil rather than empty slice", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype S struct{}\n",
		})
		s := pkg.StructByName("S")
		if s == nil {
			t.Fatalf("S missing")
		}
		if s.DocLines != nil {
			t.Fatalf("expected nil DocLines, got %v", s.DocLines)
		}
	})

	t.Run("standalone star line collapses to empty", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\n/*\nFirst.\n*\nThird.\n*/\ntype S struct{}\n",
		})
		s := pkg.StructByName("S")
		if s == nil {
			t.Fatalf("S missing")
		}
		want := []string{"", "First.", "", "Third."}
		if !slices.Equal(s.DocLines, want) {
			t.Fatalf("DocLines = %v, want %v", s.DocLines, want)
		}
	})
}

// TestPackageDocCollection verifies the converter hoists the first
// non-empty package-level doc comment found across files.
func TestPackageDocCollection(t *testing.T) {
	t.Parallel()
	t.Run("package doc surfaces on the Package node", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"doc.go": "// Package a is the canonical example.\npackage a\n",
			"a.go":   "package a\n\ntype S struct{}\n",
		})
		want := []string{"Package a is the canonical example."}
		if !slices.Equal(pkg.DocLines, want) {
			t.Fatalf("Package DocLines = %v, want %v", pkg.DocLines, want)
		}
	})

	t.Run("package without doc lines yields nil", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n",
		})
		if pkg.DocLines != nil {
			t.Fatalf("expected nil package doc, got %v", pkg.DocLines)
		}
	})
}
