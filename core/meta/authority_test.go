// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/meta"
)

func TestAuthority_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		auth meta.Authority
		want string
	}{
		{"Plugin", meta.AuthorityPlugin, "plugin"},
		{"Directive", meta.AuthorityDirective, "directive"},
		{"Manual", meta.AuthorityManual, "manual"},
		{"unknown stringifies with a marker", meta.Authority(99), "authority(99)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.auth.String(), tc.want)
		})
	}
}

func TestParseAuthority(t *testing.T) {
	t.Parallel()

	t.Run("known levels parse round-trip", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			in   string
			want meta.Authority
		}{
			{"plugin", meta.AuthorityPlugin},
			{"directive", meta.AuthorityDirective},
			{"manual", meta.AuthorityManual},
		}
		for _, tc := range cases {
			t.Run(tc.in, func(t *testing.T) {
				t.Parallel()
				got, err := meta.ParseAuthority(tc.in)
				assertNoError(t, err, "ParseAuthority")
				if got != tc.want {
					t.Fatalf("ParseAuthority(%q) = %v, want %v", tc.in, got, tc.want)
				}
			})
		}
	})

	t.Run("unknown input returns ErrUnknownAuthority", func(t *testing.T) {
		t.Parallel()
		_, err := meta.ParseAuthority("bogus")
		if !errors.Is(err, meta.ErrUnknownAuthority) {
			t.Fatalf("ParseAuthority(\"bogus\") err = %v, want ErrUnknownAuthority", err)
		}
	})
}
