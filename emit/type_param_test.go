// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestTypeParam_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindTypeParam", func(t *testing.T) {
		t.Parallel()
		var tp emit.TypeParam
		if tp.Kind() != emit.KindTypeParam {
			t.Fatalf("Kind = %s, want %s", tp.Kind(), emit.KindTypeParam)
		}
	})
}

func TestTypeParam_IsConstrained(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		tp   emit.TypeParam
		want bool
	}{
		{
			"nil Constraint is unconstrained",
			emit.TypeParam{Name: "T"},
			false,
		},
		{
			"empty Constraint is unconstrained",
			emit.TypeParam{Name: "T", Constraint: &emit.Constraint{Raw: "any"}},
			false,
		},
		{
			"named bound makes the param constrained",
			emit.TypeParam{Name: "T", Constraint: constraintFrom(externalRef("fmt", "Stringer"))},
			true,
		},
		{
			"comparable bound makes the param constrained",
			emit.TypeParam{Name: "T", Constraint: constraintFrom(builtinRef("comparable"))},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.tp.IsConstrained(); got != tc.want {
				t.Fatalf("IsConstrained() = %v, want %v (param: %+v)", got, tc.want, tc.tp)
			}
		})
	}
}
