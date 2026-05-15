// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import (
	"go.thesmos.sh/eidos/core/kind"
)

// File is one source file within a [Package]. Multi-file source
// languages (Go, TypeScript, Rust) produce one File per source file;
// single-file declarations (e.g. a single .proto schema) still
// produce a File so per-file metadata (build tags, file-level doc
// comments, imports) has a place to live.
//
// Declarations themselves live on the [Package] (not on File) — the
// model takes the package-as-flat-namespace view that most generators
// expect. Consumers that need per-file granularity can find the file
// of any node via the node's [position.Pos.File] field, or via
// [Package.Files] together with file-level [Import] / [BaseNode.DocLines].
type File struct {
	BaseNode

	// Name is the file's basename (e.g. "user.go").
	Name string `json:"name"`

	// Path is the file's full path (or repo-relative path).
	Path string `json:"path,omitempty"`

	// Imports are the import declarations in this file in source
	// order.
	Imports []*Import `json:"imports,omitempty"`

	// Owner is the [Package] this file belongs to. Populated by
	// the constructing frontend.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner *Package `json:"-"`
}

// Kind reports [KindFile].
func (*File) Kind() kind.Kind { return KindFile }

// ImportByPath returns the import with the given path, or nil when
// no such import exists. Empty path returns nil.
func (f *File) ImportByPath(path string) *Import {
	if path == "" {
		return nil
	}
	for _, imp := range f.Imports {
		if imp.Path == path {
			return imp
		}
	}
	return nil
}
