// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestInterfaceBuilder_MethodsAndEmbeds covers the interface
// surface — every nested decl wires Owner to the interface.
func TestInterfaceBuilder_MethodsAndEmbeds(t *testing.T) {
	t.Parallel()

	t.Run("methods and embeds carry interface Owner", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var i *emit.Interface
		c.Package("io", "example.com/io").
			Interface("ReadWriter", func(ib *builder.InterfaceBuilder) {
				i = ib.Node()
				ib.Method("Read", func(m *builder.MethodBuilder) {
					m.Param("p", emit.SliceOf(emit.Builtin("byte")))
					m.Return(emit.Builtin("int"))
					m.Return(emit.Builtin("error"))
				})
				ib.Embed(emit.External("io", "Writer"), nil)
			})

		if len(i.Methods) != 1 || i.Methods[0].Owner != i {
			t.Fatalf("method Owner not wired")
		}
		if len(i.Embeds) != 1 || i.Embeds[0].Owner != i {
			t.Fatalf("embed Owner not wired")
		}
	})
}

// TestInterfaceBuilder_Accessors covers the Pos / Docs / Directive /
// Target / TypeParam accessors plus an embedded EmbedBuilder
// accessor chain (interface embeds are the canonical test case for
// EmbedBuilder).
func TestInterfaceBuilder_Accessors(t *testing.T) {
	t.Parallel()

	t.Run("Pos / Docs / Directive / Target / TypeParam thread through; nested Embed accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		other := otherTarget()
		d := fixtureDirective()
		pos := fixturePos()
		var node *emit.Interface
		c.Package("p", "p").
			Interface("I", func(b *builder.InterfaceBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(other).TypeParam("T", nil).
					Embed(emit.Builtin("Reader"), func(eb *builder.EmbedBuilder) {
						eb.Pos(pos).Docs("embed").Directive(d)
					})
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != other {
			t.Fatalf("target override failed; got %v", node.Target)
		}
		if len(node.TypeParams) != 1 {
			t.Fatalf("type param not appended")
		}
		emb := node.Embeds[0]
		assertCommon(t, emb.SourcePos, emb.DocLines, emb.DirectiveList, pos, d)
	})
}
