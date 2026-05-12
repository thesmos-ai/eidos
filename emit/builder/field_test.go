// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestFieldBuilder_Accessors covers Pos / Docs / Directive plus
// the field-specific Tag and LineComment setters.
func TestFieldBuilder_Accessors(t *testing.T) {
	t.Parallel()

	t.Run("Pos / Docs / Directive / Tag / LineComment thread through", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		d := fixtureDirective()
		pos := fixturePos()
		var node *emit.Field
		c.Package("p", "p").
			Struct("S", func(sb *builder.StructBuilder) {
				sb.Field("F", emit.Builtin("int"), func(b *builder.FieldBuilder) {
					node = b.Node()
					b.Pos(pos).Docs("docs").Directive(d).Tag(`json:"f"`).LineComment("hi")
				})
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Tag != `json:"f"` {
			t.Fatalf("Tag override failed; got %q", node.Tag)
		}
		if node.LineComment != "hi" {
			t.Fatalf("LineComment override failed; got %q", node.LineComment)
		}
	})
}
