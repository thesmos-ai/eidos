// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestConstraint_IsAny(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		c    *node.Constraint
		want bool
	}{
		{"nil receiver", nil, true},
		{"empty Embedded", &node.Constraint{Raw: "any"}, true},
		{
			"with comparable embed",
			&node.Constraint{Embedded: []*node.TypeRef{namedRef("", "comparable")}},
			false,
		},
		{
			"with multiple embeds",
			&node.Constraint{Embedded: []*node.TypeRef{
				namedRef("", "comparable"),
				namedRef("fmt", "Stringer"),
			}},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.c.IsAny(); got != tc.want {
				t.Fatalf("IsAny() = %v, want %v (constraint: %+v)", got, tc.want, tc.c)
			}
		})
	}
}

func TestConstraint_IsComparable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		c    *node.Constraint
		want bool
	}{
		{"nil receiver", nil, false},
		{
			"predeclared comparable embed",
			&node.Constraint{Embedded: []*node.TypeRef{namedRef("", "comparable")}},
			true,
		},
		{
			"non-comparable embed",
			&node.Constraint{Embedded: []*node.TypeRef{namedRef("fmt", "Stringer")}},
			false,
		},
		{
			"qualified 'comparable' from a foreign package",
			&node.Constraint{Embedded: []*node.TypeRef{namedRef("other", "comparable")}},
			false,
		},
		{
			"nil entry preceding a comparable match",
			&node.Constraint{Embedded: []*node.TypeRef{nil, namedRef("", "comparable")}},
			true,
		},
		{
			"non-Named ref kind never matches",
			&node.Constraint{Embedded: []*node.TypeRef{{
				TypeKind: node.TypeRefSlice,
				Elem:     namedRef("", "comparable"),
			}}},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.c.IsComparable(); got != tc.want {
				t.Fatalf("IsComparable() = %v, want %v (constraint: %+v)", got, tc.want, tc.c)
			}
		})
	}
}
