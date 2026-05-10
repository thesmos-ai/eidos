// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

func TestProvenance_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		p    emit.Provenance
		want string
	}{
		{
			"plugin and position",
			emit.Provenance{SetBy: "validation", Pos: position.At("a.go", 5, 1)},
			"set by validation at a.go:5:1",
		},
		{
			"plugin only",
			emit.Provenance{SetBy: "audit"},
			"set by audit",
		},
		{
			"empty SetBy renders just 'set'",
			emit.Provenance{},
			"set",
		},
		{
			"empty SetBy with position renders 'set at <pos>'",
			emit.Provenance{Pos: position.At("a.go", 1, 1)},
			"set at a.go:1:1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.p.String(), tc.want)
		})
	}
}
