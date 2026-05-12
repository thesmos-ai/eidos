// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestVariableBuilder_Accessors covers the Pos / Docs / Directive
// / Target accessors on the variable builder. The Variable
// constructor itself is exercised via the [TestPackageBuilder]
// happy-path test; this stanza focuses on the accessor surface.
func TestVariableBuilder_Accessors(t *testing.T) {
	t.Parallel()

	t.Run("Pos / Docs / Directive / Target thread through", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		other := otherTarget()
		d := fixtureDirective()
		pos := fixturePos()
		var node *emit.Variable
		c.Package("p", "p").
			Variable("V", emit.Builtin("int"), nil, func(b *builder.VariableBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(other)
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != other {
			t.Fatalf("variable target override failed; got %v", node.Target)
		}
	})
}
