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
	Dir string `json:"dir,omitempty"`

	// Filename is the base filename inside Dir
	// (e.g. "user_repo_gen.go").
	Filename string `json:"filename,omitempty"`

	// Package is the package name the rendered file declares
	// (Go's `package repo`). Empty for languages without
	// package declarations at the file level.
	Package string `json:"package,omitempty"`

	// ImportPath is the canonical import path of the package the
	// rendered file lands in (e.g. "example.com/project/internal/repo").
	// Plugins set this to the source's import path when emitting
	// alongside-source, or to the consumer's centralised output
	// import path when emitting centralised; the backend forwards
	// it to the per-target [writer.ImportSet] so any [ExternalRef]
	// / [ExprExternal] whose Package matches renders bare (no
	// import, no qualifier) — the same-package elision rule.
	//
	// Empty ImportPath disables the elision; cross-package refs
	// still resolve normally. Targets distinguished only by
	// ImportPath compare unequal (Go struct equality), so plugins
	// emitting to the same logical file must agree on the value.
	ImportPath string `json:"import_path,omitempty"`
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
