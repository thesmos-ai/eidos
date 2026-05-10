// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func makeFile() *node.File {
	return &node.File{
		Name: "user.go",
		Path: "internal/users/user.go",
		Imports: []*node.Import{
			{Path: "context"},
			{Path: "github.com/example/db", Alias: "udb"},
		},
	}
}

func TestFile_ImportByPath(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching import", func(t *testing.T) {
		t.Parallel()
		got := makeFile().ImportByPath("github.com/example/db")
		if got == nil || got.Alias != "udb" {
			t.Fatalf("ImportByPath mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an empty path", func(t *testing.T) {
		t.Parallel()
		if got := makeFile().ImportByPath(""); got != nil {
			t.Fatalf("ImportByPath(\"\") should return nil; got %+v", got)
		}
	})

	t.Run("returns nil for an unknown path", func(t *testing.T) {
		t.Parallel()
		if got := makeFile().ImportByPath("missing"); got != nil {
			t.Fatalf("ImportByPath(unknown) = %+v, want nil", got)
		}
	})
}
