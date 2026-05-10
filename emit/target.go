// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

// Target identifies where the backend should write an emit entity:
// the output directory, the output filename within it, and the
// language-level package the file declares. Backends group emit
// entities by Target and render one output file per group.
//
// Two emit entities sharing a Target compose into the same file —
// the multi-generator composition path described in spec §9.
// Per-file slot injection (top, bottom, init) lives on an
// [emit.File] entity owning that Target.
//
// Target is a value type and is comparable, so it can be used
// directly as a map key for groupings.
type Target struct {
	// Dir is the output directory relative to the project root
	// (e.g. "internal/repo").
	Dir string

	// Filename is the base filename inside Dir
	// (e.g. "user_repo_gen.go").
	Filename string

	// Package is the package name the rendered file declares
	// (Go's `package repo`). Empty for languages without
	// package declarations at the file level.
	Package string
}

// IsZero reports whether t carries no routing information.
func (t Target) IsZero() bool {
	return t == Target{}
}

// JoinPath returns the file path under the project root —
// "Dir/Filename" with normalised slash separators. Returns "" when
// either component is empty.
func (t Target) JoinPath() string {
	if t.Dir == "" || t.Filename == "" {
		return ""
	}
	return t.Dir + "/" + t.Filename
}
