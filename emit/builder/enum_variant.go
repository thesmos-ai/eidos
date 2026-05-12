// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
)

// EnumVariantBuilder configures an [emit.EnumVariant] as part of an
// enum's variants. Spawned by [EnumBuilder.Variant]; the variant's
// Owner is wired by the spawning builder.
type EnumVariantBuilder struct {
	ctx *Context
	v   *emit.EnumVariant
}

// Node returns the underlying [emit.EnumVariant].
func (b *EnumVariantBuilder) Node() *emit.EnumVariant { return b.v }

// Pos overrides the variant's source position.
func (b *EnumVariantBuilder) Pos(p position.Pos) *EnumVariantBuilder {
	b.v.SourcePos = p
	return b
}

// Docs appends doc-comment lines above the variant.
func (b *EnumVariantBuilder) Docs(lines ...string) *EnumVariantBuilder {
	b.v.DocLines = append(b.v.DocLines, lines...)
	return b
}

// Directive attaches d to the variant's directive list.
func (b *EnumVariantBuilder) Directive(d *directive.Directive) *EnumVariantBuilder {
	b.v.DirectiveList = append(b.v.DirectiveList, d)
	return b
}
