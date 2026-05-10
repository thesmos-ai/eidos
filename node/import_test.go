// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestImport_LocalName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		imp  node.Import
		want string
	}{
		{"alias overrides path", node.Import{Path: "github.com/foo/bar", Alias: "buzz"}, "buzz"},
		{"derives from path's last segment", node.Import{Path: "github.com/foo/bar"}, "bar"},
		{"single-segment path returns the segment", node.Import{Path: "context"}, "context"},
		{"empty path returns empty", node.Import{}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.imp.LocalName(), tc.want)
		})
	}
}
