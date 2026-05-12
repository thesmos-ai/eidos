// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// FileBuilder configures an [emit.File] as part of a
// [PackageBuilder]'s accumulating package. Spawned by
// [PackageBuilder.File].
//
// In typical plugin code an [emit.File] is obtained from the live
// store via `store.EmitView.FileFor(target)` rather than constructed
// fresh — files are deduplicated by Target across plugins. Use
// FileBuilder when the plugin is building a Package from scratch
// (e.g. in tests, or in a plugin that wholly owns its outputs);
// reach for the slot-append API on [Context] when contributing into
// an existing file.
type FileBuilder struct {
	ctx *Context
	f   *emit.File
}

// File appends an [emit.File] to the package. The target arg controls
// the file's Dir / Name / Package fields; pass `emit.Target{}` to
// inherit the spawning [Context]'s default target.
func (b *PackageBuilder) File(target emit.Target, fn func(*FileBuilder)) *PackageBuilder {
	if target.IsZero() {
		target = b.ctx.target
	}
	f := &emit.File{
		Name:    target.Filename,
		Package: target.Package,
		Dir:     target.Dir,
		Owner:   b.pkg,
	}
	fb := &FileBuilder{ctx: b.ctx, f: f}
	if fn != nil {
		fn(fb)
	}
	b.pkg.Files = append(b.pkg.Files, f)
	return b
}

// Node returns the underlying [emit.File].
func (b *FileBuilder) Node() *emit.File { return b.f }

// Pos overrides the file's source position.
func (b *FileBuilder) Pos(p position.Pos) *FileBuilder {
	b.f.SourcePos = p
	return b
}

// Docs appends doc-comment lines above the file's `package` clause.
func (b *FileBuilder) Docs(lines ...string) *FileBuilder {
	b.f.DocLines = append(b.f.DocLines, lines...)
	return b
}

// Import appends a direct import to the file. fn (which may be nil)
// configures the import's alias and per-import slots.
func (b *FileBuilder) Import(path string, fn func(*ImportBuilder)) *FileBuilder {
	imp := &emit.Import{Path: path, Owner: b.f}
	if fn != nil {
		fn(&ImportBuilder{ctx: b.ctx, i: imp})
	}
	b.f.Imports = append(b.f.Imports, imp)
	return b
}

// BlankImport is the shorthand for a side-effect-only import
// (`import _ "..."`). Equivalent to
// `b.Import(path, func(i *ImportBuilder) { i.Alias("_") })`.
func (b *FileBuilder) BlankImport(path string) *FileBuilder {
	return b.Import(path, func(i *ImportBuilder) { i.Alias("_") })
}
