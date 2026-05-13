// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// FieldBuilder configures an [emit.Field] as part of a struct's
// fields. Spawned by [StructBuilder.Field]; the field's Owner is
// wired by the spawning builder so handlers never touch the
// back-pointer.
type FieldBuilder struct {
	ctx *Context
	f   *emit.Field
}

// Node returns the underlying [emit.Field].
func (b *FieldBuilder) Node() *emit.Field { return b.f }

// Pos overrides the field's source position.
func (b *FieldBuilder) Pos(p position.Pos) *FieldBuilder {
	b.f.SourcePos = p
	return b
}

// Origin records the source node the field was lifted from so
// downstream consumers (backend render-site lookups, the
// `explain` command) can reach the source-side meta bag.
func (b *FieldBuilder) Origin(n node.Node) *FieldBuilder {
	b.f.OriginNode = n
	return b
}

// Docs appends doc-comment lines above the field. Renderers add the
// `// ` prefix; pass the raw textual lines.
func (b *FieldBuilder) Docs(lines ...string) *FieldBuilder {
	b.f.DocLines = append(b.f.DocLines, lines...)
	return b
}

// Directive attaches d to the field's directive list.
func (b *FieldBuilder) Directive(d *directive.Directive) *FieldBuilder {
	b.f.DirectiveList = append(b.f.DirectiveList, d)
	return b
}

// Tag sets the directly-declared struct tag, without enclosing
// backticks (e.g. `json:"id" validate:"required"`). Cross-cutting
// generators may add further tag entries via the field's tags slot —
// use [Context.AppendTag] for the canonical append shape.
func (b *FieldBuilder) Tag(tag string) *FieldBuilder {
	b.f.Tag = tag
	return b
}

// LineComment sets the trailing comment rendered after the field's
// type (and tag, if any). Empty omits the comment entirely.
func (b *FieldBuilder) LineComment(comment string) *FieldBuilder {
	b.f.LineComment = comment
	return b
}
