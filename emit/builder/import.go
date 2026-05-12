// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// ImportBuilder configures an [emit.Import] — one import declaration
// on a file. Spawned by [FileBuilder.Import] (or
// [FileBuilder.BlankImport] for the side-effect-only shorthand); the
// import's Owner is wired by the spawning builder.
type ImportBuilder struct {
	ctx *Context
	i   *emit.Import
}

// Node returns the underlying [emit.Import].
func (b *ImportBuilder) Node() *emit.Import { return b.i }

// Pos overrides the import's source position.
func (b *ImportBuilder) Pos(p position.Pos) *ImportBuilder {
	b.i.SourcePos = p
	return b
}

// Docs appends doc-comment lines above the import.
func (b *ImportBuilder) Docs(lines ...string) *ImportBuilder {
	b.i.DocLines = append(b.i.DocLines, lines...)
	return b
}

// Directive attaches d to the import's directive list.
func (b *ImportBuilder) Directive(d *directive.Directive) *ImportBuilder {
	b.i.DirectiveList = append(b.i.DirectiveList, d)
	return b
}

// Alias sets the local alias for the import. Pass "_" for a
// side-effect-only import (Go's `import _ "..."`); use
// [FileBuilder.BlankImport] for that shorthand.
func (b *ImportBuilder) Alias(alias string) *ImportBuilder {
	b.i.Alias = alias
	return b
}
