// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/opt"
)

func TestFieldKind_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind opt.FieldKind
		want string
	}{
		{"String", opt.KindString, "string"},
		{"Int", opt.KindInt, "int"},
		{"Bool", opt.KindBool, "bool"},
		{"StringList", opt.KindStringList, "string_list"},
		{"Duration", opt.KindDuration, "duration"},
		{"unknown stringifies with a marker", opt.FieldKind(99), "kind(99)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.kind.String(), tc.want)
		})
	}
}
