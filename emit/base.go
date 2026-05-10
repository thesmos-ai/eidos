// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
)

// BaseEmit supplies the fields and methods every concrete emit type
// shares. Each concrete type embeds BaseEmit by value and overrides
// [Kind] to return its [directive.Kind] constant.
//
// All struct fields are exported so generators and tests can populate
// them via struct literals. The MetaBag field is allocated lazily on
// first call to [BaseEmit.Meta] so struct-literal construction works
// without an explicit constructor invocation.
//
// During the generator phase the emit tree is mutable; once the
// constructing generator returns, downstream consumers (later
// generators reading prior emit and the backend) treat the tree as
// frozen — see the spec mutability contract.
type BaseEmit struct {
	// SourcePos is the source position this emit value reflects.
	// Frontends synthesising emit purely from plugin logic should
	// use [position.Synthetic] tags.
	SourcePos position.Pos

	// DocLines holds doc comment text — either preserved from the
	// originating source node or freshly authored by the generator.
	DocLines []string

	// DirectiveList holds the directives attached to this emit
	// value (typically copied verbatim from the originating source
	// node so backend rendering can re-read them).
	DirectiveList []*directive.Directive

	// MetaBag is the lazily-allocated metadata storage. Access via
	// [BaseEmit.Meta] rather than touching the field directly.
	MetaBag *meta.Bag

	// OriginNode is the source [node.Node] this emit value was
	// derived from. nil for purely-generated artifacts.
	OriginNode node.Node
}

// Pos returns [BaseEmit.SourcePos].
func (b *BaseEmit) Pos() position.Pos { return b.SourcePos }

// Docs returns [BaseEmit.DocLines]. The returned slice aliases
// internal storage; callers must not mutate it.
func (b *BaseEmit) Docs() []string { return b.DocLines }

// Directives returns [BaseEmit.DirectiveList]. The returned slice
// aliases internal storage; callers must not mutate it.
func (b *BaseEmit) Directives() []*directive.Directive { return b.DirectiveList }

// Directive returns the first directive whose [directive.Name]
// matches name, or nil when no such directive is attached.
func (b *BaseEmit) Directive(name directive.Name) *directive.Directive {
	for _, d := range b.DirectiveList {
		if d.Name == name {
			return d
		}
	}
	return nil
}

// HasDirective reports whether at least one directive with the given
// name is attached.
func (b *BaseEmit) HasDirective(name directive.Name) bool {
	return b.Directive(name) != nil
}

// Meta returns the metadata bag for this emit value, allocating one
// on first access. The allocation is one-shot per value; the bag is
// concurrent-safe via its own internal lock.
func (b *BaseEmit) Meta() *meta.Bag {
	if b.MetaBag == nil {
		b.MetaBag = meta.NewBag()
	}
	return b.MetaBag
}

// Origin returns [BaseEmit.OriginNode] — the source node this emit
// value was derived from, or nil for purely-generated artifacts.
func (b *BaseEmit) Origin() node.Node { return b.OriginNode }
