// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestDeclBuilders_MetadataAccessors exercises the Pos / Docs /
// Directive / Target / Node accessors every decl-level builder
// shares. The mirror-shape across builders makes them a natural
// table fixture: one row per builder kind, one set of assertions.
func TestDeclBuilders_MetadataAccessors(t *testing.T) {
	t.Parallel()

	d := &directive.Directive{Name: directive.Name("gen:probe")}
	pos := position.Pos{File: "fixture.go", Line: 10}
	otherTarget := emit.Target{Dir: "other", Filename: "other.go", Package: "other"}

	t.Run("struct accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.Struct
		c.Package("p", "p").
			Struct("S", func(b *builder.StructBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(otherTarget).TypeParam("T", nil)
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != otherTarget {
			t.Fatalf("target override failed; got %v", node.Target)
		}
		if len(node.TypeParams) != 1 || node.TypeParams[0].Name != "T" {
			t.Fatalf("type param not appended; got %+v", node.TypeParams)
		}
	})

	t.Run("interface accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.Interface
		c.Package("p", "p").
			Interface("I", func(b *builder.InterfaceBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(otherTarget).TypeParam("T", nil).
					Embed(emit.Builtin("Reader"), func(eb *builder.EmbedBuilder) {
						eb.Pos(pos).Docs("embed").Directive(d)
					})
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != otherTarget {
			t.Fatalf("target override failed; got %v", node.Target)
		}
		if len(node.TypeParams) != 1 {
			t.Fatalf("type param not appended")
		}
		emb := node.Embeds[0]
		assertCommon(t, emb.SourcePos, emb.DocLines, emb.DirectiveList, pos, d)
	})

	t.Run("function accessors and body", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		body := emit.NewRawStmt(`return nil`)
		var node *emit.Function
		c.Package("p", "p").
			Function("F", func(b *builder.FunctionBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(otherTarget).
					TypeParam("T", nil).Body(body)
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != otherTarget {
			t.Fatalf("target override failed")
		}
		if len(node.TypeParams) != 1 || len(node.Body) != 1 {
			t.Fatalf("type param / body mis-applied; %+v %+v", node.TypeParams, node.Body)
		}
	})

	t.Run("method accessors and body", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		body := emit.NewRawStmt(`return nil`)
		var node *emit.Method
		c.Package("p", "p").
			Struct("S", func(sb *builder.StructBuilder) {
				sb.Method("M", func(b *builder.MethodBuilder) {
					node = b.Node()
					b.Pos(pos).Docs("docs").Directive(d).TypeParam("T", nil).Body(body)
				})
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if len(node.TypeParams) != 1 || len(node.Body) != 1 {
			t.Fatalf("method type param / body mis-applied")
		}
	})

	t.Run("enum accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.Enum
		c.Package("p", "p").
			Enum("E", emit.Builtin("int"), func(b *builder.EnumBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(otherTarget).
					Variant("V", nil, func(vb *builder.EnumVariantBuilder) {
						vb.Pos(pos).Docs("vd").Directive(d)
					})
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != otherTarget {
			t.Fatalf("enum target override failed")
		}
		v := node.Variants[0]
		assertCommon(t, v.SourcePos, v.DocLines, v.DirectiveList, pos, d)
	})

	t.Run("alias accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.Alias
		c.Package("p", "p").
			Alias("A", emit.Builtin("string"), func(b *builder.AliasBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).File(otherTarget).TypeParam("T", nil)
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.File != otherTarget {
			t.Fatalf("alias File override failed; got %v", node.File)
		}
		if len(node.TypeParams) != 1 {
			t.Fatalf("type param not appended")
		}
	})

	t.Run("variable accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.Variable
		c.Package("p", "p").
			Variable("V", emit.Builtin("int"), nil, func(b *builder.VariableBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(otherTarget)
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != otherTarget {
			t.Fatalf("variable target override failed")
		}
	})

	t.Run("constant accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.Constant
		c.Package("p", "p").
			Constant("C", emit.Builtin("int"), nil, func(b *builder.ConstantBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Directive(d).Target(otherTarget)
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
		if node.Target != otherTarget {
			t.Fatalf("constant target override failed")
		}
	})

	t.Run("field accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.Field
		c.Package("p", "p").
			Struct("S", func(sb *builder.StructBuilder) {
				sb.Field("F", emit.Builtin("int"), func(b *builder.FieldBuilder) {
					node = b.Node()
					b.Pos(pos).Docs("docs").Directive(d)
				})
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
	})

	t.Run("param accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.Param
		c.Package("p", "p").
			Function("F", func(fb *builder.FunctionBuilder) {
				fb.Param("p", emit.Builtin("int"), func(b *builder.ParamBuilder) {
					node = b.Node()
					b.Pos(pos).Docs("docs").Directive(d)
				})
			})
		assertCommon(t, node.SourcePos, node.DocLines, node.DirectiveList, pos, d)
	})

	t.Run("file accessors", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		var node *emit.File
		c.Package("p", "p").
			File(otherTarget, func(b *builder.FileBuilder) {
				node = b.Node()
				b.Pos(pos).Docs("docs").Import("fmt", func(ib *builder.ImportBuilder) {
					ib.Pos(pos).Docs("imp").Directive(d).Alias("fmtalias")
				})
			})
		if node.SourcePos != pos {
			t.Fatalf("file Pos override failed; got %v", node.SourcePos)
		}
		if len(node.DocLines) != 1 {
			t.Fatalf("file docs missing")
		}
		imp := node.Imports[0]
		assertCommon(t, imp.SourcePos, imp.DocLines, imp.DirectiveList, pos, d)
		if imp.Alias != "fmtalias" {
			t.Fatalf("import alias = %q, want fmtalias", imp.Alias)
		}
	})

	t.Run("package docs accessor", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		pkg, err := c.Package("p", "p").Docs("package docs").Build()
		if err != nil {
			t.Fatalf("Build returned %v", err)
		}
		if len(pkg.DocLines) != 1 || pkg.DocLines[0] != "package docs" {
			t.Fatalf("package docs not appended; got %+v", pkg.DocLines)
		}
	})
}

// assertCommon asserts the Pos / Docs / Directive trio every
// decl-level builder shares — single helper keeps the per-builder
// test stanzas focused on the bits that vary.
func assertCommon(
	t *testing.T,
	gotPos position.Pos,
	gotDocs []string,
	gotDirs []*directive.Directive,
	wantPos position.Pos,
	wantDir *directive.Directive,
) {
	t.Helper()
	if gotPos != wantPos {
		t.Fatalf("Pos = %v, want %v", gotPos, wantPos)
	}
	if len(gotDocs) != 1 {
		t.Fatalf("expected one doc line; got %+v", gotDocs)
	}
	if len(gotDirs) != 1 || gotDirs[0] != wantDir {
		t.Fatalf("expected the supplied directive in DirectiveList; got %+v", gotDirs)
	}
}
