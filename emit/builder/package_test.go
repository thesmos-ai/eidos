// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
)

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
		if pkg.Name != "users" || pkg.Path != "example.com/users" {
			t.Fatalf("pkg identity wrong; got %q / %q", pkg.Name, pkg.Path)
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
			t.Fatalf("expected one Alias + one NamedType; got %d", len(pkg.Aliases))
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
		if pkg.Structs[0].Target != defaultTarget {
			t.Fatalf("struct target = %v, want %v", pkg.Structs[0].Target, defaultTarget)
		}
		if pkg.Functions[0].Target != defaultTarget {
			t.Fatalf("function target = %v, want %v", pkg.Functions[0].Target, defaultTarget)
		}
	})
}

// TestPackageBuilder_DocsAndNode covers the [builder.PackageBuilder]
// docs and Node accessors.
func TestPackageBuilder_DocsAndNode(t *testing.T) {
	t.Parallel()

	t.Run("Docs appends lines on the underlying package", func(t *testing.T) {
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

	t.Run("Node returns the underlying *emit.Package", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		b := c.Package("p", "p")
		if b.Node() == nil {
			t.Fatalf("Node returned nil")
		}
	})

	t.Run("Err mirrors Build's error component", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		// A clean Package returns no error.
		b := c.Package("p", "p")
		if got := b.Err(); got != nil {
			t.Fatalf("Err on clean package should be nil; got %v", got)
		}
	})
}

// TestPackageBuilder_Method covers the top-level Method
// constructor. The decl lands on [emit.Package.Methods] (not
// nested under a Struct/Interface/Alias); the Anchor's default
// origin is stamped as the method's Owner so the framework's
// downstream routing and rewire passes can resolve the receiver
// type. OwnerRef is populated in lock-step.
func TestPackageBuilder_Method(t *testing.T) {
	t.Parallel()

	t.Run("Method appends to Package.Methods", func(t *testing.T) {
		t.Parallel()
		anchor := &node.Enum{Name: "Status", Package: "example.com/store"}
		pkg, err := builder.For("enum").Anchor(anchor).
			Method("String", nil).
			Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got := len(pkg.Methods); got != 1 {
			t.Fatalf("Methods len = %d, want 1", got)
		}
		if got := pkg.Methods[0].Name; got != "String" {
			t.Fatalf("Methods[0].Name = %q, want %q", got, "String")
		}
	})

	t.Run("Method stamps Owner from Anchor's default origin", func(t *testing.T) {
		t.Parallel()
		anchor := &node.Enum{Name: "Status", Package: "example.com/store"}
		pkg, err := builder.For("enum").Anchor(anchor).
			Method("String", nil).
			Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		m := pkg.Methods[0]
		if m.Owner == nil {
			t.Fatalf("Owner not stamped")
		}
		if got, want := m.Owner.OwnerName(), "Status"; got != want {
			t.Fatalf("Owner.OwnerName = %q, want %q", got, want)
		}
	})

	t.Run("Method stamps OwnerRef in lock-step with Owner", func(t *testing.T) {
		t.Parallel()
		anchor := &node.Enum{Name: "Status", Package: "example.com/store"}
		pkg, err := builder.For("enum").Anchor(anchor).
			Method("String", nil).
			Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		m := pkg.Methods[0]
		if m.OwnerRef.IsZero() {
			t.Fatalf("OwnerRef not stamped")
		}
		if got, want := m.OwnerRef.QName, "example.com/store.Status"; got != want {
			t.Fatalf("OwnerRef.QName = %q, want %q", got, want)
		}
	})

	t.Run("Method sets Package to the anchored package path", func(t *testing.T) {
		t.Parallel()
		anchor := &node.Enum{Name: "Status", Package: "example.com/store"}
		pkg, err := builder.For("enum").Anchor(anchor).
			Method("String", nil).
			Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if got, want := pkg.Methods[0].Package, "example.com/store"; got != want {
			t.Fatalf("Method.Package = %q, want %q", got, want)
		}
	})

	t.Run("MethodBuilder Receiver / Body / Returns flow through", func(t *testing.T) {
		t.Parallel()
		anchor := &node.Enum{Name: "Status", Package: "example.com/store"}
		pkg, err := builder.For("enum").Anchor(anchor).
			Method("String", func(m *builder.MethodBuilder) {
				m.Receiver("e", emit.External("example.com/store", "Status"))
				m.Return(emit.Builtin("string"))
				m.Body(emit.NewReturn(emit.NewLiteralString("active")))
			}).
			Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		m := pkg.Methods[0]
		if m.ReceiverName != "e" || m.Receiver == nil {
			t.Fatalf("receiver not threaded; name=%q type=%v", m.ReceiverName, m.Receiver)
		}
		if len(m.Returns) != 1 || len(m.Body) != 1 {
			t.Fatalf("returns/body not threaded: returns=%d body=%d", len(m.Returns), len(m.Body))
		}
	})
}
