// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture_test

import (
	"testing"

	"go.thesmos.sh/eidos/storefixture"
	"go.thesmos.sh/eidos/node"
)

func TestNamed(t *testing.T) {
	t.Parallel()

	t.Run("produces a Named ref with no package", func(t *testing.T) {
		t.Parallel()
		r := asNamedRef(t, storefixture.Named("string"))
		if r.Name != "string" || r.Package != "" {
			t.Fatalf("Named ref shape wrong: %+v", r)
		}
		if !r.IsBuiltin() {
			t.Fatalf("Named with no package should be a builtin")
		}
	})
}

func TestPkgNamed(t *testing.T) {
	t.Parallel()

	t.Run("produces a Named ref with package and name", func(t *testing.T) {
		t.Parallel()
		r := asNamedRef(t, storefixture.PkgNamed("context", "Context"))
		if r.Package != "context" || r.Name != "Context" {
			t.Fatalf("PkgNamed ref shape wrong: %+v", r)
		}
		if r.IsBuiltin() {
			t.Fatalf("PkgNamed should not be a builtin")
		}
	})
}

func TestWithArgs(t *testing.T) {
	t.Parallel()

	t.Run("returns a generic ref carrying the supplied type args", func(t *testing.T) {
		t.Parallel()
		base := storefixture.PkgNamed("foo", "Map")
		r := storefixture.WithArgs(base, storefixture.Named("string"), storefixture.Named("int"))
		if !r.IsGeneric() {
			t.Fatalf("WithArgs result should be generic; got %+v", r)
		}
		if len(r.TypeArgs) != 2 || r.TypeArgs[0].Name != "string" || r.TypeArgs[1].Name != "int" {
			t.Fatalf("type args wrong: %+v", r.TypeArgs)
		}
		if base.IsGeneric() {
			t.Fatalf("WithArgs must not mutate its input; base became generic: %+v", base)
		}
	})

	t.Run("panics for nil ref", func(t *testing.T) {
		t.Parallel()
		requirePanic(t, func() {
			storefixture.WithArgs(nil, storefixture.Named("string"))
		})
	})

	t.Run("panics for non-Named ref", func(t *testing.T) {
		t.Parallel()
		requirePanic(t, func() {
			storefixture.WithArgs(storefixture.Slice(storefixture.Named("string")))
		})
	})
}

func TestPointer(t *testing.T) {
	t.Parallel()

	t.Run("produces a Pointer ref over elem", func(t *testing.T) {
		t.Parallel()
		r := storefixture.Pointer(storefixture.Named("int"))
		if !r.IsPointer() {
			t.Fatalf("expected pointer ref, got %+v", r)
		}
		if r.Elem == nil || r.Elem.Name != "int" {
			t.Fatalf("pointer element wrong: %+v", r.Elem)
		}
	})
}

func TestSlice(t *testing.T) {
	t.Parallel()

	t.Run("produces a Slice ref over elem", func(t *testing.T) {
		t.Parallel()
		r := storefixture.Slice(storefixture.Named("byte"))
		if !r.IsSlice() {
			t.Fatalf("expected slice ref, got %+v", r)
		}
		if r.Elem == nil || r.Elem.Name != "byte" {
			t.Fatalf("slice element wrong: %+v", r.Elem)
		}
	})
}

func TestArray(t *testing.T) {
	t.Parallel()

	t.Run("produces an Array ref with element and length", func(t *testing.T) {
		t.Parallel()
		r := storefixture.Array(storefixture.Named("int"), 8)
		if !r.IsArray() {
			t.Fatalf("expected array ref, got %+v", r)
		}
		if r.ArrayLen != 8 {
			t.Fatalf("array length wrong: got %d, want 8", r.ArrayLen)
		}
		if r.Elem == nil || r.Elem.Name != "int" {
			t.Fatalf("array element wrong: %+v", r.Elem)
		}
	})
}

func TestMap(t *testing.T) {
	t.Parallel()

	t.Run("produces a Map ref with key and value types", func(t *testing.T) {
		t.Parallel()
		r := storefixture.Map(storefixture.Named("string"), storefixture.Named("int"))
		if !r.IsMap() {
			t.Fatalf("expected map ref, got %+v", r)
		}
		if r.MapKey == nil || r.MapKey.Name != "string" {
			t.Fatalf("map key wrong: %+v", r.MapKey)
		}
		if r.MapValue == nil || r.MapValue.Name != "int" {
			t.Fatalf("map value wrong: %+v", r.MapValue)
		}
	})
}

func TestTypeParamRef(t *testing.T) {
	t.Parallel()

	t.Run("produces a TypeParam-kind ref carrying the parameter name", func(t *testing.T) {
		t.Parallel()
		r := storefixture.TypeParamRef("T")
		if !r.IsTypeParam() {
			t.Fatalf("expected TypeParam ref, got %+v", r)
		}
		if r.Name != "T" {
			t.Fatalf("Name = %q, want %q", r.Name, "T")
		}
	})
}

func TestAnonStruct(t *testing.T) {
	t.Parallel()

	t.Run("produces an AnonStruct ref with fields wired to the ref", func(t *testing.T) {
		t.Parallel()
		field := &node.Field{Name: "ID", Type: storefixture.Named("string")}
		embed := &node.Embed{Type: storefixture.PkgNamed("io", "Reader")}
		r := storefixture.AnonStruct([]*node.Field{field}, []*node.Embed{embed})
		if !r.IsAnonStruct() {
			t.Fatalf("expected AnonStruct ref, got %+v", r)
		}
		if field.Owner != r {
			t.Fatalf("Field.Owner should be wired to the AnonStruct ref")
		}
		if embed.Owner != r {
			t.Fatalf("Embed.Owner should be wired to the AnonStruct ref")
		}
	})
}

func TestAnonInterface(t *testing.T) {
	t.Parallel()

	t.Run("produces an AnonInterface ref with methods wired to the ref", func(t *testing.T) {
		t.Parallel()
		method := &node.Method{Name: "Read"}
		embed := &node.Embed{Type: storefixture.PkgNamed("io", "Reader")}
		r := storefixture.AnonInterface([]*node.Method{method}, []*node.Embed{embed})
		if !r.IsAnonInterface() {
			t.Fatalf("expected AnonInterface ref, got %+v", r)
		}
		if method.Owner != r {
			t.Fatalf("Method.Owner should be wired to the AnonInterface ref")
		}
		if embed.Owner != r {
			t.Fatalf("Embed.Owner should be wired to the AnonInterface ref")
		}
	})
}

func TestConstraint(t *testing.T) {
	t.Parallel()

	t.Run("produces a Constraint with the supplied embeds", func(t *testing.T) {
		t.Parallel()
		c := storefixture.Constraint(
			storefixture.PkgNamed("fmt", "Stringer"),
			storefixture.Named("comparable"),
		)
		if c.IsAny() {
			t.Fatalf("constraint with embeds should not be IsAny")
		}
		if !c.IsComparable() {
			t.Fatalf("constraint should reflect comparable bound")
		}
		if len(c.Embedded) != 2 {
			t.Fatalf("expected 2 embeds, got %d", len(c.Embedded))
		}
	})

	t.Run("zero embeds produces an IsAny constraint", func(t *testing.T) {
		t.Parallel()
		c := storefixture.Constraint()
		if !c.IsAny() {
			t.Fatalf("constraint with no embeds should be IsAny")
		}
	})
}

func TestFunc(t *testing.T) {
	t.Parallel()

	t.Run("produces a Func ref with params and returns", func(t *testing.T) {
		t.Parallel()
		params := []*node.TypeRef{storefixture.Named("string")}
		returns := []*node.TypeRef{storefixture.Named("int"), storefixture.Named("error")}
		r := storefixture.Func(params, returns)
		if !r.IsFunc() {
			t.Fatalf("expected func ref, got %+v", r)
		}
		if len(r.FuncParams) != 1 || r.FuncParams[0].Name != "string" {
			t.Fatalf("func params wrong: %+v", r.FuncParams)
		}
		switch {
		case len(r.FuncReturns) != 2,
			r.FuncReturns[0].Name != "int",
			r.FuncReturns[1].Name != "error":
			t.Fatalf("func returns wrong: %+v", r.FuncReturns)
		}
	})
}
