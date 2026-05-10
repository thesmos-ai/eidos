// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/opt"
)

func makeSchema() opt.Schema {
	return opt.Schema{
		Fields: []opt.Field{
			{Name: "alpha", Kind: opt.KindString},
			{Name: "beta", Kind: opt.KindInt},
		},
	}
}

func TestSchema_Lookup(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching field", func(t *testing.T) {
		t.Parallel()
		s := makeSchema()
		got, ok := s.Lookup("beta")
		if !ok {
			t.Fatalf("Lookup should find beta")
		}
		if got.Kind != opt.KindInt {
			t.Fatalf("Lookup kind = %v, want Int", got.Kind)
		}
	})

	t.Run("returns false for an unknown name", func(t *testing.T) {
		t.Parallel()
		s := makeSchema()
		if _, ok := s.Lookup("missing"); ok {
			t.Fatalf("Lookup should be false for an unknown name")
		}
	})
}

func TestSchema_HasField(t *testing.T) {
	t.Parallel()

	t.Run("returns true for a declared field", func(t *testing.T) {
		t.Parallel()
		if !makeSchema().HasField("alpha") {
			t.Fatalf("HasField should be true for alpha")
		}
	})

	t.Run("returns false for an undeclared field", func(t *testing.T) {
		t.Parallel()
		if makeSchema().HasField("zeta") {
			t.Fatalf("HasField should be false for zeta")
		}
	})
}

func TestSchema_Names(t *testing.T) {
	t.Parallel()

	t.Run("returns field names in declaration order", func(t *testing.T) {
		t.Parallel()
		got := makeSchema().Names()
		want := []string{"alpha", "beta"}
		if !slices.Equal(got, want) {
			t.Fatalf("Names = %v, want %v", got, want)
		}
	})
}
