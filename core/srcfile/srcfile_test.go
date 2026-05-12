// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package srcfile_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/core/srcfile"
)

// TestWithSuffix covers the stringer-style filename derivation
// across the supported input shapes: absolute / relative source
// paths, the empty-pos fallback, and the no-extension edge case.
func TestWithSuffix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		pos      position.Pos
		fallback string
		suffix   string
		want     string
	}{
		{
			name:     "absolute path strips dir + extension",
			pos:      position.Pos{File: "/abs/users/article.go"},
			fallback: "article",
			suffix:   "_repo.go",
			want:     "article_repo.go",
		},
		{
			name:     "relative path behaves the same",
			pos:      position.Pos{File: "users/article.go"},
			fallback: "article",
			suffix:   "_builder.go",
			want:     "article_builder.go",
		},
		{
			name:     "non-go suffix passes through verbatim",
			pos:      position.Pos{File: "src/lib.rs"},
			fallback: "lib",
			suffix:   "_codegen.rs",
			want:     "lib_codegen.rs",
		},
		{
			name:     "empty pos falls back to lower-cased name + suffix",
			pos:      position.Pos{},
			fallback: "Article",
			suffix:   "_repo.go",
			want:     "article_repo.go",
		},
		{
			name:     "extension-less basename uses bare name",
			pos:      position.Pos{File: "/abs/Makefile"},
			fallback: "Makefile",
			suffix:   "_gen",
			want:     "Makefile_gen",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := srcfile.WithSuffix(tc.pos, tc.fallback, tc.suffix); got != tc.want {
				t.Fatalf("WithSuffix = %q, want %q", got, tc.want)
			}
		})
	}
}
