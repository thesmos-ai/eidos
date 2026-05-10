// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
)

// recordingVisitor collects the directive.Kind of every node Walk
// visits, in visit order. Tests assert on the resulting slice.
type recordingVisitor struct {
	kinds *[]directive.Kind
}

func (r recordingVisitor) Visit(n node.Node) node.Visitor {
	*r.kinds = append(*r.kinds, n.Kind())
	return r
}

func recordWalk(n node.Node) []directive.Kind {
	var kinds []directive.Kind
	node.Walk(n, recordingVisitor{kinds: &kinds})
	return kinds
}

func TestWalk_NilNodeIsNoop(t *testing.T) {
	t.Parallel()

	t.Run("nil node yields no visits", func(t *testing.T) {
		t.Parallel()
		var kinds []directive.Kind
		node.Walk(nil, recordingVisitor{kinds: &kinds})
		if len(kinds) != 0 {
			t.Fatalf("nil node should not be visited; got %v", kinds)
		}
	})

	t.Run("nil visitor yields no visits", func(t *testing.T) {
		t.Parallel()
		// The Walk call returns immediately; nothing observable to
		// assert beyond not panicking.
		node.Walk(&node.Struct{}, nil)
	})
}

func TestWalk_PrunedSubtree(t *testing.T) {
	t.Parallel()

	t.Run("returning nil from Visit stops descent", func(t *testing.T) {
		t.Parallel()
		var visited int
		pruning := node.VisitorFunc(func(_ node.Node) node.Visitor {
			visited++
			return nil
		})
		s := &node.Struct{
			Fields:  []*node.Field{{Name: "ID"}},
			Methods: []*node.Method{{Name: "Save"}},
		}
		node.Walk(s, pruning)
		if visited != 1 {
			t.Fatalf("expected 1 visit (root only); got %d", visited)
		}
	})
}

func TestWalk_PackageDescentOrder(t *testing.T) {
	t.Parallel()

	t.Run("visits files, imports, then declarations of every kind in order", func(t *testing.T) {
		t.Parallel()
		p := &node.Package{
			Files:      []*node.File{{Name: "user.go"}},
			Imports:    []*node.Import{{Path: "context"}},
			Structs:    []*node.Struct{{Name: "User"}},
			Interfaces: []*node.Interface{{Name: "Repo"}},
			Functions:  []*node.Function{{Name: "Open"}},
			Variables:  []*node.Variable{{Name: "Default"}},
			Constants:  []*node.Constant{{Name: "Pi"}},
			Enums:      []*node.Enum{{Name: "Status"}},
			Aliases:    []*node.Alias{{Name: "ID"}},
		}
		got := recordWalk(p)
		want := []directive.Kind{
			node.KindPackage,
			node.KindFile,
			node.KindImport,
			node.KindStruct,
			node.KindInterface,
			node.KindFunction,
			node.KindVariable,
			node.KindConstant,
			node.KindEnum,
			node.KindAlias,
		}
		if !slices.Equal(got, want) {
			t.Fatalf("visit order = %v, want %v", got, want)
		}
	})
}

func TestWalk_StructDescent(t *testing.T) {
	t.Parallel()

	t.Run("visits TypeParams, Fields, Embeds, Methods", func(t *testing.T) {
		t.Parallel()
		s := &node.Struct{
			TypeParams: []*node.TypeParam{{Name: "T"}},
			Fields:     []*node.Field{{Name: "ID", Type: namedRef("", "string")}},
			Embeds:     []*node.Embed{{Type: namedRef("", "Base")}},
			Methods:    []*node.Method{{Name: "Save"}},
		}
		got := recordWalk(s)
		want := []directive.Kind{
			node.KindStruct, node.KindTypeParam, node.KindField, node.KindTypeRef,
			node.KindEmbed, node.KindTypeRef, node.KindMethod,
		}
		if !slices.Equal(got, want) {
			t.Fatalf("visit order = %v, want %v", got, want)
		}
	})
}

func TestWalk_InterfaceDescent(t *testing.T) {
	t.Parallel()

	t.Run("visits TypeParams, Methods, Embeds", func(t *testing.T) {
		t.Parallel()
		i := &node.Interface{
			TypeParams: []*node.TypeParam{{Name: "T"}},
			Methods:    []*node.Method{{Name: "Get"}},
			Embeds:     []*node.Embed{{Type: namedRef("", "Base")}},
		}
		got := recordWalk(i)
		want := []directive.Kind{
			node.KindInterface, node.KindTypeParam, node.KindMethod, node.KindEmbed, node.KindTypeRef,
		}
		if !slices.Equal(got, want) {
			t.Fatalf("visit order = %v, want %v", got, want)
		}
	})
}

func TestWalk_MethodDescent(t *testing.T) {
	t.Parallel()

	t.Run("visits TypeParams, Params, Returns", func(t *testing.T) {
		t.Parallel()
		m := &node.Method{
			TypeParams: []*node.TypeParam{{Name: "T"}},
			Params:     []*node.Param{{Type: namedRef("", "int")}},
			Returns:    []*node.TypeRef{namedRef("", "error")},
		}
		got := recordWalk(m)
		want := []directive.Kind{
			node.KindMethod, node.KindTypeParam, node.KindParam, node.KindTypeRef, node.KindTypeRef,
		}
		if !slices.Equal(got, want) {
			t.Fatalf("visit order = %v, want %v", got, want)
		}
	})
}

func TestWalk_FunctionDescent(t *testing.T) {
	t.Parallel()

	t.Run("visits TypeParams, Params, Returns", func(t *testing.T) {
		t.Parallel()
		f := &node.Function{
			TypeParams: []*node.TypeParam{{Name: "T"}},
			Params:     []*node.Param{{Type: namedRef("", "int")}},
			Returns:    []*node.TypeRef{namedRef("", "error")},
		}
		got := recordWalk(f)
		want := []directive.Kind{
			node.KindFunction, node.KindTypeParam, node.KindParam, node.KindTypeRef, node.KindTypeRef,
		}
		if !slices.Equal(got, want) {
			t.Fatalf("visit order = %v, want %v", got, want)
		}
	})
}

func TestWalk_EnumDescent(t *testing.T) {
	t.Parallel()

	t.Run("visits Underlying then Variants", func(t *testing.T) {
		t.Parallel()
		e := &node.Enum{
			Underlying: namedRef("", "int"),
			Variants:   []*node.EnumVariant{{Name: "A"}, {Name: "B"}},
		}
		got := recordWalk(e)
		want := []directive.Kind{
			node.KindEnum, node.KindTypeRef, node.KindEnumVariant, node.KindEnumVariant,
		}
		if !slices.Equal(got, want) {
			t.Fatalf("visit order = %v, want %v", got, want)
		}
	})
}

func TestWalk_AliasDescent(t *testing.T) {
	t.Parallel()

	t.Run("visits TypeParams then Target", func(t *testing.T) {
		t.Parallel()
		a := &node.Alias{
			TypeParams: []*node.TypeParam{{Name: "T"}},
			Target:     namedRef("", "int"),
		}
		got := recordWalk(a)
		want := []directive.Kind{node.KindAlias, node.KindTypeParam, node.KindTypeRef}
		if !slices.Equal(got, want) {
			t.Fatalf("visit order = %v, want %v", got, want)
		}
	})
}

func TestWalk_TypeRefVariants(t *testing.T) {
	t.Parallel()

	t.Run("Pointer/Slice/Array/Chan descend into Elem", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			ref  *node.TypeRef
		}{
			{"Pointer", &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: namedRef("", "int")}},
			{"Slice", &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: namedRef("", "int")}},
			{"Array", &node.TypeRef{TypeKind: node.TypeRefArray, ArrayLen: 3, Elem: namedRef("", "int")}},
			{"Chan", &node.TypeRef{TypeKind: node.TypeRefChan, Elem: namedRef("", "int")}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				got := recordWalk(tc.ref)
				if !slices.Equal(got, []directive.Kind{node.KindTypeRef, node.KindTypeRef}) {
					t.Fatalf("expected two TypeRef visits; got %v", got)
				}
			})
		}
	})

	t.Run("Map visits MapKey and MapValue", func(t *testing.T) {
		t.Parallel()
		r := &node.TypeRef{
			TypeKind: node.TypeRefMap,
			MapKey:   namedRef("", "string"),
			MapValue: namedRef("", "int"),
		}
		got := recordWalk(r)
		if len(got) != 3 {
			t.Fatalf("expected three visits (root + key + value); got %v", got)
		}
	})

	t.Run("Func visits parameters and returns", func(t *testing.T) {
		t.Parallel()
		r := &node.TypeRef{
			TypeKind:    node.TypeRefFunc,
			FuncParams:  []*node.TypeRef{namedRef("", "int")},
			FuncReturns: []*node.TypeRef{namedRef("", "error")},
		}
		got := recordWalk(r)
		if len(got) != 3 {
			t.Fatalf("expected three visits (root + param + return); got %v", got)
		}
	})

	t.Run("Named visits TypeArgs", func(t *testing.T) {
		t.Parallel()
		r := namedRef("", "Container")
		r.TypeArgs = []*node.TypeRef{namedRef("", "int"), namedRef("", "string")}
		got := recordWalk(r)
		if len(got) != 3 {
			t.Fatalf("expected three visits (root + 2 type args); got %v", got)
		}
	})
}

func TestWalk_LeafKinds(t *testing.T) {
	t.Parallel()

	t.Run("Variable, Constant, Embed, Field, Param descend into their type", func(t *testing.T) {
		t.Parallel()
		ref := namedRef("", "int")
		cases := []node.Node{
			&node.Variable{Type: ref},
			&node.Constant{Type: ref},
			&node.Embed{Type: ref},
			&node.Field{Type: ref},
			&node.Param{Type: ref},
		}
		for _, n := range cases {
			got := recordWalk(n)
			if len(got) != 2 {
				t.Fatalf("%s expected to visit itself + Type; got %v", n.Kind(), got)
			}
		}
	})

	t.Run("TypeParam descends into Constraint when present", func(t *testing.T) {
		t.Parallel()
		tp := &node.TypeParam{Name: "T", Constraint: namedRef("", "any")}
		got := recordWalk(tp)
		if len(got) != 2 {
			t.Fatalf("expected TypeParam + Constraint; got %v", got)
		}
	})

	t.Run("EnumVariant is a leaf node", func(t *testing.T) {
		t.Parallel()
		got := recordWalk(&node.EnumVariant{Name: "Active"})
		if len(got) != 1 || got[0] != node.KindEnumVariant {
			t.Fatalf("EnumVariant should be a leaf; got %v", got)
		}
	})

	t.Run("File visits its Imports", func(t *testing.T) {
		t.Parallel()
		f := &node.File{Imports: []*node.Import{{Path: "context"}, {Path: "fmt"}}}
		got := recordWalk(f)
		want := []directive.Kind{node.KindFile, node.KindImport, node.KindImport}
		if !slices.Equal(got, want) {
			t.Fatalf("visit order = %v, want %v", got, want)
		}
	})
}
