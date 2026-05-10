// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestImport_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindImport", func(t *testing.T) {
		t.Parallel()
		var i emit.Import
		if i.Kind() != emit.KindImport {
			t.Fatalf("Kind = %s, want %s", i.Kind(), emit.KindImport)
		}
	})
}

func TestImport_LocalName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		path   string
		alias  string
		want   string
		reason string
	}{
		{"alias overrides path-derived name", "context", "ctxalias", "ctxalias", ""},
		{"path-derived name uses last segment", "github.com/foo/bar/baz", "", "baz", ""},
		{"single-segment path returns the path itself", "context", "", "context", ""},
		{"empty path returns empty", "", "", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			imp := &emit.Import{Path: tc.path, Alias: tc.alias}
			assertEqualString(t, imp.LocalName(), tc.want)
		})
	}
}

func TestImport_IsBlank(t *testing.T) {
	t.Parallel()

	t.Run("returns true for underscore alias", func(t *testing.T) {
		t.Parallel()
		imp := &emit.Import{Path: "_/anything", Alias: "_"}
		if !imp.IsBlank() {
			t.Fatalf("underscore alias should be blank")
		}
	})

	t.Run("returns false for any other alias", func(t *testing.T) {
		t.Parallel()
		imp := &emit.Import{Path: "context", Alias: "ctx"}
		if imp.IsBlank() {
			t.Fatalf("named alias should not be blank")
		}
	})
}
