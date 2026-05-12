// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// EmbedBuilder configures an [emit.Embed] — an embedded type in a
// struct or interface. Spawned by [StructBuilder.Embed] or
// [InterfaceBuilder.Embed]; the embed's Owner is wired by the
// spawning builder.
type EmbedBuilder struct {
	ctx *Context
	e   *emit.Embed
}

// Node returns the underlying [emit.Embed].
func (b *EmbedBuilder) Node() *emit.Embed { return b.e }

// Pos overrides the embed's source position.
func (b *EmbedBuilder) Pos(p position.Pos) *EmbedBuilder {
	b.e.SourcePos = p
	return b
}

// Docs appends doc-comment lines above the embedded type.
func (b *EmbedBuilder) Docs(lines ...string) *EmbedBuilder {
	b.e.DocLines = append(b.e.DocLines, lines...)
	return b
}

// Directive attaches d to the embed's directive list.
func (b *EmbedBuilder) Directive(d *directive.Directive) *EmbedBuilder {
	b.e.DirectiveList = append(b.e.DirectiveList, d)
	return b
}
