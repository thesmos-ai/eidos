// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestTypeParam_IsConstrained(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		tp   node.TypeParam
		want bool
	}{
		{
			"nil Constraint is unconstrained",
			node.TypeParam{},
			false,
		},
		{
			"empty Constraint is unconstrained",
			node.TypeParam{Constraint: &node.Constraint{Raw: "any"}},
			false,
		},
		{
			"named bound makes the param constrained",
			node.TypeParam{Constraint: constraintFrom(namedRef("fmt", "Stringer"))},
			true,
		},
		{
			"comparable bound makes the param constrained",
			node.TypeParam{Constraint: constraintFrom(namedRef("", "comparable"))},
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
