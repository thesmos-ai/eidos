// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestTarget_IsZero(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		t    emit.Target
		want bool
	}{
		{"zero value is zero", emit.Target{}, true},
		{"non-zero Dir is not zero", emit.Target{Dir: "x"}, false},
		{"non-zero Filename is not zero", emit.Target{Filename: "x.go"}, false},
		{"non-zero Package is not zero", emit.Target{Package: "x"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.t.IsZero() != tc.want {
				t.Fatalf("IsZero %+v = %v, want %v", tc.t, tc.t.IsZero(), tc.want)
			}
		})
	}
}

func TestTarget_JoinPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		t    emit.Target
		want string
	}{
		{"both populated", emit.Target{Dir: "internal/repo", Filename: "user_gen.go"}, "internal/repo/user_gen.go"},
		{"empty dir yields empty", emit.Target{Filename: "user.go"}, ""},
		{"empty filename yields empty", emit.Target{Dir: "internal/repo"}, ""},
		{"both empty yields empty", emit.Target{}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.t.JoinPath(), tc.want)
		})
	}
}

func TestTarget_IsComparable(t *testing.T) {
	t.Parallel()

	t.Run("equal Targets compare as map keys", func(t *testing.T) {
		t.Parallel()
		m := map[emit.Target]int{}
		k := emit.Target{Dir: "a", Filename: "b.go", Package: "b"}
		m[k] = 1
		m[emit.Target{Dir: "a", Filename: "b.go", Package: "b"}] = 2
		if len(m) != 1 {
			t.Fatalf("Target should be comparable as a map key; got %d entries", len(m))
		}
	})
}
