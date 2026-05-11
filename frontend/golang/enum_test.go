// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"testing"

	"go.thesmos.sh/eidos/frontend/golang"
)

// TestDetectEnums covers the typed-iota → Enum promotion path.
func TestDetectEnums(t *testing.T) {
	t.Parallel()
	t.Run("typed iota constants promote to an Enum", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Status int\n\nconst (\n\tStatusActive Status = iota\n\tStatusInactive\n\tStatusArchived\n)\n",
		})
		e := pkg.EnumByName("Status")
		if e == nil {
			t.Fatalf("Status not promoted to enum")
		}
		if len(e.Variants) != 3 {
			t.Fatalf("expected 3 variants, got %d", len(e.Variants))
		}
		if e.Underlying == nil || e.Underlying.Name != "int" {
			t.Fatalf("Enum.Underlying = %+v, want int", e.Underlying)
		}
		// Variant iota values preserved via MetaIotaValue.
		for i, v := range e.Variants {
			got, _ := golang.MetaIotaValue.Get(v.Meta())
			if got != i {
				t.Fatalf("variant %d MetaIotaValue = %v, want %d", i, got, i)
			}
		}
	})

	t.Run("absorbed alias is removed from the Aliases slice", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\ntype Status int\n\nconst (\n\tStatusActive Status = iota\n)\n",
		})
		if pkg.AliasByName("Status") != nil {
			t.Fatalf("Status alias should have been absorbed by enum promotion")
		}
	})

	t.Run("typed constants without a matching alias stay as Constants", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\nconst Limit int = 10\n",
		})
		if pkg.EnumByName("int") != nil {
			t.Fatalf("must not promote primitives into enum")
		}
		if pkg.ConstantByName("Limit") == nil {
			t.Fatalf("Limit should remain as a Constant entry")
		}
	})

	t.Run("untyped constants are never promoted", func(t *testing.T) {
		t.Parallel()
		pkg := requirePackage(t, map[string]string{
			"a.go": "package a\n\nconst (\n\tA = iota\n\tB\n\tC\n)\n",
		})
		if len(pkg.Enums) != 0 {
			t.Fatalf("untyped iota constants must not produce enums")
		}
	})
}
