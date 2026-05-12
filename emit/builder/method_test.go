// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestMethodBuilder_ParamCallbackAndNamedReturn covers the
// optional Param-callback and the named-Return forms on Method —
// these are method-specific because struct/interface/alias methods
// all flow through the same MethodBuilder.
func TestMethodBuilder_ParamCallbackAndNamedReturn(t *testing.T) {
	t.Parallel()

	t.Run("Param with a callback runs it; named Return threads the name", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var m *emit.Method
		c.Package("p", "p").
			Struct("S", func(sb *builder.StructBuilder) {
				sb.Method("M", func(mb *builder.MethodBuilder) {
					m = mb.Node()
					mb.Param("opts", emit.Builtin("Options"), func(pb *builder.ParamBuilder) {
						pb.Variadic()
					})
					mb.Return(emit.Builtin("int"), "n")
				})
			})
		if !m.Params[0].Variadic {
			t.Fatalf("variadic flag not threaded through method.Param callback")
		}
		if m.Returns[0].Name != "n" {
			t.Fatalf("named return not threaded; got %+v", m.Returns)
		}
	})
}

// TestMethodBuilder_Accessors covers the Pos / Docs / Directive /
// TypeParam / Body / Receiver accessors. Methods inherit their
// target from the host decl rather than carrying their own.
func TestMethodBuilder_Accessors(t *testing.T) {
	t.Parallel()

	t.Run("Pos / Docs / Directive / TypeParam / Body / Receiver thread through", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		d := fixtureDirective()
		pos := fixturePos()
		body := emit.NewRawStmt(`return nil`)
		var node *emit.Method
		c.Package("p", "p").
			Struct("S", func(sb *builder.StructBuilder) {
				sb.Method("M", func(b *builder.MethodBuilder) {
					node = b.Node()
					b.Pos(pos).
						Docs("docs").
						Directive(d).
						Receiver("r", emit.Builtin("R")).
						TypeParam("T", nil).
						Body(body)
				})
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if len(node.TypeParams) != 1 || len(node.Body) != 1 {
			t.Fatalf("method type param / body mis-applied")
		}
		if node.ReceiverName != "r" || node.Receiver == nil {
			t.Fatalf("receiver not threaded; got name=%q type=%v", node.ReceiverName, node.Receiver)
		}
	})
}
