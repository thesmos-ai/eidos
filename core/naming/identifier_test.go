// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package naming_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/naming"
)

func TestIdentifier(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty input becomes underscore", "", "_"},
		{"plain identifier passes through", "userID", "userID"},
		{"underscores are preserved", "user_id", "user_id"},
		{"non-letter non-digit becomes underscore", "user-id.v2", "user_id_v2"},
		{"leading digit gets underscore prefix", "2things", "_2things"},
		{"digit elsewhere is left alone", "thing2things", "thing2things"},
		{"unicode letters are preserved", "héllo_wörld", "héllo_wörld"},
		{"all-symbol input becomes all-underscore", "!!!", "___"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, naming.Identifier(tc.in), tc.want)
		})
	}
}
