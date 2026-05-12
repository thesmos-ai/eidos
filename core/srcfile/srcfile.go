// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package srcfile derives the per-source-file output filename
// reference plugins use when emitting alongside the source — the
// stringer-style convention `<src-basename>_<suffix>`, where
// `<src-basename>` is the source's filename without the `.go`
// extension. Two source structs in `article.go` annotated with
// `+gen:builder` therefore both target `article_builder.go`, so
// multiple per-struct decls compose into one rendered file.
package srcfile

import (
	"path/filepath"
	"strings"

	"go.thesmos.sh/eidos/core/position"
)

// WithSuffix returns the per-source-file output filename for a
// source entity at pos. The returned name strips the `.go`
// extension from the source's basename and appends suffix.
//
// pos.File may be empty (synthetic source, missing position) — the
// fallback identifier is used in that case to keep the rendered
// filename stable and predictable.
//
// Example: WithSuffix(Pos{File: "/abs/users/article.go"},
// "article", "_repo.go") returns "article_repo.go".
func WithSuffix(pos position.Pos, fallback, suffix string) string {
	if pos.File != "" {
		base := filepath.Base(pos.File)
		if ext := filepath.Ext(base); ext != "" {
			base = base[:len(base)-len(ext)]
		}
		if base != "" {
			return base + suffix
		}
	}
	return strings.ToLower(fallback) + suffix
}
