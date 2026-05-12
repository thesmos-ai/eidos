// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// defaultTarget is the canonical target used by the test fixtures.
// Centralised so an audit of which file the fixtures emit into reads
// from a single place.
var defaultTarget = emit.Target{Dir: "users", Filename: "users.go", Package: "users"}

// TestContext_ForBindsIdentityAndTarget covers the foundational
// [builder.For] constructor: the Context exposes the bound plugin
// identity and target unchanged, and provenance helpers stamp the
// same plugin name.
func TestContext_ForBindsIdentityAndTarget(t *testing.T) {
	t.Parallel()

	t.Run("SetBy and Target return the constructor arguments", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		if got := c.SetBy(); got != "repogen" {
			t.Fatalf("SetBy = %q, want %q", got, "repogen")
		}
		if got := c.Target(); got != defaultTarget {
			t.Fatalf("Target = %v, want %v", got, defaultTarget)
		}
	})

	t.Run("Provenance stamps SetBy and (optionally) ID", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		bare := c.Provenance()
		if bare.SetBy != "repogen" || bare.ID != "" {
			t.Fatalf("Provenance() = %+v; want SetBy=repogen ID=empty", bare)
		}
		withID := c.Provenance("seed")
		if withID.SetBy != "repogen" || withID.ID != "seed" {
			t.Fatalf("Provenance(\"seed\") = %+v; want SetBy=repogen ID=seed", withID)
		}
	})

	t.Run("WithTarget returns a copy without mutating the receiver", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		other := emit.Target{Dir: "audit", Filename: "audit.go", Package: "users"}
		c2 := c.WithTarget(other)
		if c.Target() != defaultTarget {
			t.Fatalf("receiver should be untouched; got %v", c.Target())
		}
		if c2.Target() != other {
			t.Fatalf("clone should carry the new target; got %v", c2.Target())
		}
	})
}

// TestPackageBuilder_AllDeclKindsLand covers every decl kind the
// [builder.PackageBuilder] exposes: each appends to the underlying
// package, inherits the context target, and surfaces through
// [builder.PackageBuilder.Build].
func TestPackageBuilder_AllDeclKindsLand(t *testing.T) {
	t.Parallel()

	t.Run("each decl method appends one entity to the package", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		pkg, err := c.Package("users", "example.com/users").
			Struct("S", nil).
			Interface("I", nil).
			Function("F", nil).
			Enum("E", emit.Builtin("int"), nil).
			Alias("A", emit.Builtin("string"), nil).
			NamedType("N", emit.Builtin("int"), nil).
			Variable("V", emit.Builtin("int"), nil, nil).
			Constant("C", emit.Builtin("int"),
				&emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitInt, RawText: "1"}, nil).
			Build()
		if err != nil {
			t.Fatalf("Build returned %v", err)
		}

		if got := pkg.Name; got != "users" {
			t.Fatalf("pkg.Name = %q, want users", got)
		}
		if got := pkg.Path; got != "example.com/users" {
			t.Fatalf("pkg.Path = %q, want example.com/users", got)
		}
		if len(pkg.Structs) != 1 || pkg.Structs[0].Name != "S" {
			t.Fatalf("expected one struct S; got %+v", pkg.Structs)
		}
		if len(pkg.Interfaces) != 1 || pkg.Interfaces[0].Name != "I" {
			t.Fatalf("expected one interface I; got %+v", pkg.Interfaces)
		}
		if len(pkg.Functions) != 1 || pkg.Functions[0].Name != "F" {
			t.Fatalf("expected one function F; got %+v", pkg.Functions)
		}
		if len(pkg.Enums) != 1 || pkg.Enums[0].Name != "E" {
			t.Fatalf("expected one enum E; got %+v", pkg.Enums)
		}
		if len(pkg.Aliases) != 2 {
			t.Fatalf("expected one Alias + one NamedType = 2 aliases; got %d", len(pkg.Aliases))
		}
		if !pkg.Aliases[0].IsAlias {
			t.Fatalf("first alias should be a true alias")
		}
		if pkg.Aliases[1].IsAlias {
			t.Fatalf("second alias (NamedType) should not be a true alias")
		}
		if len(pkg.Variables) != 1 || pkg.Variables[0].Name != "V" {
			t.Fatalf("expected one variable V; got %+v", pkg.Variables)
		}
		if len(pkg.Constants) != 1 || pkg.Constants[0].Name != "C" {
			t.Fatalf("expected one constant C; got %+v", pkg.Constants)
		}
	})

	t.Run("decls inherit the context target", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		pkg, err := c.Package("users", "example.com/users").
			Struct("S", nil).
			Function("F", nil).
			Build()
		if err != nil {
			t.Fatalf("Build returned %v", err)
		}
		if got := pkg.Structs[0].Target; got != defaultTarget {
			t.Fatalf("struct target = %v, want %v", got, defaultTarget)
		}
		if got := pkg.Functions[0].Target; got != defaultTarget {
			t.Fatalf("function target = %v, want %v", got, defaultTarget)
		}
	})
}

// TestStructBuilder_NestedShape covers the nested decl-construction
// surface: fields, methods, embeds, and type parameters wire Owner
// back-pointers automatically.
func TestStructBuilder_NestedShape(t *testing.T) {
	t.Parallel()

	t.Run("Field / Method / Embed / TypeParam wire Owner back-pointers", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var s *emit.Struct
		c.Package("users", "example.com/users").
			Struct("User", func(sb *builder.StructBuilder) {
				s = sb.Node()
				sb.Field("ID", emit.Builtin("string"), nil)
				sb.Field("Email", emit.Builtin("string"), func(f *builder.FieldBuilder) {
					f.Tag(`json:"email"`).LineComment("user email")
				})
				sb.Embed(emit.Internal(s), nil)
				sb.Method("Validate", func(m *builder.MethodBuilder) {
					m.Receiver("u", emit.Ptr(emit.Internal(s)))
					m.Return(emit.Builtin("error"))
				})
				sb.TypeParam("T", nil)
			})

		if len(s.Fields) != 2 {
			t.Fatalf("expected 2 fields, got %d", len(s.Fields))
		}
		for _, f := range s.Fields {
			if f.Owner != s {
				t.Fatalf("field %q Owner = %v, want %p", f.Name, f.Owner, s)
			}
		}
		if s.Fields[1].Tag != `json:"email"` {
			t.Fatalf("field tag not threaded; got %q", s.Fields[1].Tag)
		}
		if s.Fields[1].LineComment != "user email" {
			t.Fatalf("line comment not threaded; got %q", s.Fields[1].LineComment)
		}
		if len(s.Methods) != 1 || s.Methods[0].Owner != s {
			t.Fatalf("method Owner = %v, want %p", s.Methods[0].Owner, s)
		}
		if len(s.Embeds) != 1 || s.Embeds[0].Owner != s {
			t.Fatalf("embed Owner = %v, want %p", s.Embeds[0].Owner, s)
		}
		if len(s.TypeParams) != 1 || s.TypeParams[0].Owner != s {
			t.Fatalf("type-param Owner = %v, want %p", s.TypeParams[0].Owner, s)
		}
	})
}

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
			t.Fatalf("method Owner = %v, want %p", i.Methods[0].Owner, i)
		}
		if len(i.Embeds) != 1 || i.Embeds[0].Owner != i {
			t.Fatalf("embed Owner = %v, want %p", i.Embeds[0].Owner, i)
		}
	})
}

// TestFunctionBuilder_ParamsAndReturns covers the function surface.
// Params wire Owner; Return appends accumulate.
func TestFunctionBuilder_ParamsAndReturns(t *testing.T) {
	t.Parallel()

	t.Run("variadic param marks the underlying Param.Variadic", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var f *emit.Function
		c.Package("io", "example.com/io").
			Function("Sum", func(fb *builder.FunctionBuilder) {
				f = fb.Node()
				fb.Param("nums", emit.Builtin("int"), func(pb *builder.ParamBuilder) {
					pb.Variadic()
				})
				fb.Return(emit.Builtin("int"))
			})

		if len(f.Params) != 1 || !f.Params[0].Variadic {
			t.Fatalf("expected variadic param; got %+v", f.Params)
		}
		if f.Params[0].Owner != f {
			t.Fatalf("param Owner = %v, want %p", f.Params[0].Owner, f)
		}
		if len(f.Returns) != 1 {
			t.Fatalf("expected one return slot; got %d", len(f.Returns))
		}
	})

	t.Run("Return accepts an optional name and stamps it on the slot", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var f *emit.Function
		c.Package("io", "example.com/io").
			Function("F", func(fb *builder.FunctionBuilder) {
				f = fb.Node()
				fb.Return(emit.Builtin("int"), "result")
				fb.Return(emit.Builtin("error"), "err")
			})
		if len(f.Returns) != 2 || f.Returns[0].Name != "result" || f.Returns[1].Name != "err" {
			t.Fatalf("named returns not threaded; got %+v", f.Returns)
		}
	})
}

// TestMethodBuilder_OptionalParamCallbackAndNamedReturn covers the
// optional Param-callback and the named-Return forms on Method — the
// canonical happy-path test exercises neither.
func TestMethodBuilder_OptionalParamCallbackAndNamedReturn(t *testing.T) {
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

// TestAliasBuilder_MethodWithCallbackRuns covers the `fn != nil`
// branch of [AliasBuilder.Method] — the callback runs and Owner is
// wired to the alias.
func TestAliasBuilder_MethodWithCallbackRuns(t *testing.T) {
	t.Parallel()

	t.Run("non-nil method callback runs with method-builder", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var captured *emit.Method
		c.Package("p", "p").
			NamedType("UserID", emit.Builtin("string"), func(ab *builder.AliasBuilder) {
				ab.Method("Validate", func(mb *builder.MethodBuilder) {
					captured = mb.Node()
					mb.Return(emit.Builtin("error"))
				})
			})
		if captured == nil || len(captured.Returns) != 1 {
			t.Fatalf("method callback did not run; got %+v", captured)
		}
	})
}

// TestAliasBuilder_MethodOnTrueAliasRecordsError covers the
// language-rule guard: methods are not allowed on `type X = Y`
// aliases. The violation accumulates onto the parent
// [builder.PackageBuilder] and surfaces via
// [builder.PackageBuilder.Build], leaving the resulting graph clean
// of the offending method.
func TestAliasBuilder_MethodOnTrueAliasRecordsError(t *testing.T) {
	t.Parallel()

	t.Run("Method on a true alias records ErrAliasMethodForbidden", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var a *emit.Alias
		pkg, err := c.Package("users", "example.com/users").
			Alias("UserID", emit.Builtin("string"), func(ab *builder.AliasBuilder) {
				a = ab.Node()
				ab.Method("Validate", nil)
			}).
			Build()
		if !errors.Is(err, builder.ErrAliasMethodForbidden) {
			t.Fatalf("expected ErrAliasMethodForbidden; got %v", err)
		}
		if len(a.Methods) != 0 {
			t.Fatalf("method should be dropped on a true alias; got %d methods", len(a.Methods))
		}
		if len(pkg.Aliases) != 1 {
			t.Fatalf("alias itself should still land in the package; got %d", len(pkg.Aliases))
		}
	})

	t.Run("Method on a named-type alias succeeds", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var a *emit.Alias
		_, err := c.Package("users", "example.com/users").
			NamedType("UserID", emit.Builtin("string"), func(ab *builder.AliasBuilder) {
				a = ab.Node()
				ab.Method("Validate", nil)
			}).
			Build()
		if err != nil {
			t.Fatalf("Build returned %v", err)
		}
		if len(a.Methods) != 1 || a.Methods[0].Owner != a {
			t.Fatalf("expected one method owned by the alias; got %+v", a.Methods)
		}
	})
}

// TestEnumBuilder_VariantsCarryOwner covers the enum-variant Owner
// back-pointer wiring.
func TestEnumBuilder_VariantsCarryOwner(t *testing.T) {
	t.Parallel()

	t.Run("variants carry enum Owner", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var e *emit.Enum
		c.Package("status", "example.com/status").
			Enum("State", emit.Builtin("int"), func(eb *builder.EnumBuilder) {
				e = eb.Node()
				eb.Variant("Active", nil, nil)
				eb.Variant("Inactive", nil, nil)
			})
		if len(e.Variants) != 2 {
			t.Fatalf("expected 2 variants; got %d", len(e.Variants))
		}
		for _, v := range e.Variants {
			if v.Owner != e {
				t.Fatalf("variant %q Owner = %v, want %p", v.Name, v.Owner, e)
			}
		}
	})
}

// TestFileBuilder_ImportsAndBlanks covers the file builder's import
// surface, including the BlankImport shorthand.
func TestFileBuilder_ImportsAndBlanks(t *testing.T) {
	t.Parallel()

	t.Run("Import and BlankImport append owner-wired imports", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var file *emit.File
		c.Package("users", "example.com/users").
			File(emit.Target{}, func(fb *builder.FileBuilder) {
				file = fb.Node()
				fb.Import("fmt", nil)
				fb.BlankImport("embed")
			})
		if len(file.Imports) != 2 {
			t.Fatalf("expected 2 imports; got %d", len(file.Imports))
		}
		if file.Imports[0].Path != "fmt" || file.Imports[0].Alias != "" {
			t.Fatalf("first import wrong; got %+v", file.Imports[0])
		}
		if file.Imports[1].Path != "embed" || file.Imports[1].Alias != "_" {
			t.Fatalf("blank import should carry alias _; got %+v", file.Imports[1])
		}
		for _, imp := range file.Imports {
			if imp.Owner != file {
				t.Fatalf("import Owner = %v, want %p", imp.Owner, file)
			}
		}
	})

	t.Run("zero target on File() inherits the Context target", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		var file *emit.File
		c.Package("users", "example.com/users").
			File(emit.Target{}, func(fb *builder.FileBuilder) {
				file = fb.Node()
			})
		if file.Dir != defaultTarget.Dir || file.Name != defaultTarget.Filename {
			t.Fatalf("file target not inherited; got Dir=%q Name=%q", file.Dir, file.Name)
		}
	})
}
