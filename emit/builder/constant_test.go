// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestConstantBuilder_Accessors covers the Pos / Docs / Directive
// / Target accessors on the constant builder. The Constant
// constructor itself is exercised via the [TestPackageBuilder]
// happy-path test; this stanza focuses on the accessor surface.
func TestConstantBuilder_Accessors(t *testing.T) {
	t.Parallel()

	t.Run("Pos / Docs / Directive / Target thread through", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		other := otherTarget()
		d := fixtureDirective()
		pos := fixturePos()
		var node *emit.Constant
		c.Package("p", "p").
			Constant("C", emit.Builtin("int"), nil, func(b *builder.ConstantBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(other)
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != other {
			t.Fatalf("constant target override failed; got %v", node.Target)
		}
	})
}
