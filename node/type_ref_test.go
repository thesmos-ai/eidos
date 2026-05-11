// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func TestTypeRefKind_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		k    node.TypeRefKind
		want string
	}{
		{"Named", node.TypeRefNamed, "named"},
		{"Pointer", node.TypeRefPointer, "pointer"},
		{"Slice", node.TypeRefSlice, "slice"},
		{"Array", node.TypeRefArray, "array"},
		{"Map", node.TypeRefMap, "map"},
		{"Func", node.TypeRefFunc, "func"},
		{"TypeParam", node.TypeRefTypeParam, "type_param"},
		{"AnonStruct", node.TypeRefAnonStruct, "anon_struct"},
		{"AnonInterface", node.TypeRefAnonInterface, "anon_interface"},
		{"unknown stringifies with a marker", node.TypeRefKind(99), "type_ref_kind(?)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.k.String(), tc.want)
		})
	}
}

func TestTypeRef_IsBuiltin(t *testing.T) {
	t.Parallel()

	t.Run("Named with empty package is builtin", func(t *testing.T) {
		t.Parallel()
		if !namedRef("", "int").IsBuiltin() {
			t.Fatalf("Named with empty package should be builtin")
		}
	})

	t.Run("Named with package is not builtin", func(t *testing.T) {
		t.Parallel()
		if namedRef("github.com/foo/bar", "User").IsBuiltin() {
			t.Fatalf("Named with package should not be builtin")
		}
	})

	t.Run("non-Named ref is not builtin", func(t *testing.T) {
		t.Parallel()
		if (&node.TypeRef{TypeKind: node.TypeRefPointer}).IsBuiltin() {
			t.Fatalf("Pointer ref should not be builtin")
		}
	})
}

func TestTypeRef_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("Named with TypeArgs is generic", func(t *testing.T) {
		t.Parallel()
		r := namedRef("", "Container")
		r.TypeArgs = []*node.TypeRef{namedRef("", "int")}
		if !r.IsGeneric() {
			t.Fatalf("Named with TypeArgs should be generic")
		}
	})

	t.Run("Named without TypeArgs is not generic", func(t *testing.T) {
		t.Parallel()
		if namedRef("", "int").IsGeneric() {
			t.Fatalf("Named without TypeArgs should not be generic")
		}
	})
}

func TestTypeRef_KindPredicates(t *testing.T) {
	t.Parallel()

	predicates := []struct {
		name string
		kind node.TypeRefKind
		pred func(*node.TypeRef) bool
	}{
		{"IsPointer", node.TypeRefPointer, (*node.TypeRef).IsPointer},
		{"IsSlice", node.TypeRefSlice, (*node.TypeRef).IsSlice},
		{"IsArray", node.TypeRefArray, (*node.TypeRef).IsArray},
		{"IsMap", node.TypeRefMap, (*node.TypeRef).IsMap},
		{"IsFunc", node.TypeRefFunc, (*node.TypeRef).IsFunc},
		{"IsTypeParam", node.TypeRefTypeParam, (*node.TypeRef).IsTypeParam},
		{"IsAnonStruct", node.TypeRefAnonStruct, (*node.TypeRef).IsAnonStruct},
		{"IsAnonInterface", node.TypeRefAnonInterface, (*node.TypeRef).IsAnonInterface},
	}

	kinds := []node.TypeRefKind{
		node.TypeRefNamed,
		node.TypeRefPointer,
		node.TypeRefSlice,
		node.TypeRefArray,
		node.TypeRefMap,
		node.TypeRefFunc,
		node.TypeRefTypeParam,
		node.TypeRefAnonStruct,
		node.TypeRefAnonInterface,
	}

	for _, k := range kinds {
		for _, p := range predicates {
			name := p.name + "_on_" + k.String()
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				ref := &node.TypeRef{TypeKind: k}
				want := k == p.kind
				if got := p.pred(ref); got != want {
					t.Fatalf("%s on %s = %v, want %v", p.name, k, got, want)
				}
			})
		}
	}
}

func TestTypeRef_Equal(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver and nil arg are equal", func(t *testing.T) {
		t.Parallel()
		var a, b *node.TypeRef
		if !a.Equal(b) {
			t.Fatalf("nil should equal nil")
		}
	})

	t.Run("nil receiver against non-nil is not equal", func(t *testing.T) {
		t.Parallel()
		var a *node.TypeRef
		if a.Equal(namedRef("", "int")) {
			t.Fatalf("nil should not equal non-nil")
		}
	})

	t.Run("non-nil receiver against nil is not equal", func(t *testing.T) {
		t.Parallel()
		if namedRef("", "int").Equal(nil) {
			t.Fatalf("non-nil should not equal nil")
		}
	})

	t.Run("identical Named refs are equal", func(t *testing.T) {
		t.Parallel()
		if !namedRef("ctx", "Context").Equal(namedRef("ctx", "Context")) {
			t.Fatalf("identical named refs should be equal")
		}
	})

	t.Run("Named refs with different packages are not equal", func(t *testing.T) {
		t.Parallel()
		if namedRef("a", "T").Equal(namedRef("b", "T")) {
			t.Fatalf("different packages should not be equal")
		}
	})

	t.Run("Named refs with different names are not equal", func(t *testing.T) {
		t.Parallel()
		if namedRef("p", "A").Equal(namedRef("p", "B")) {
			t.Fatalf("different names should not be equal")
		}
	})

	t.Run("different kinds are not equal", func(t *testing.T) {
		t.Parallel()
		ptr := &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: namedRef("", "int")}
		if namedRef("", "int").Equal(ptr) {
			t.Fatalf("Named and Pointer should not be equal")
		}
	})

	t.Run("Pointer refs compare by Elem", func(t *testing.T) {
		t.Parallel()
		a := &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: namedRef("", "int")}
		b := &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: namedRef("", "int")}
		c := &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: namedRef("", "string")}
		if !a.Equal(b) {
			t.Fatalf("pointers with same Elem should be equal")
		}
		if a.Equal(c) {
			t.Fatalf("pointers with different Elem should not be equal")
		}
	})

	t.Run("Slice refs compare by Elem", func(t *testing.T) {
		t.Parallel()
		a := &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: namedRef("", "byte")}
		b := &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: namedRef("", "byte")}
		if !a.Equal(b) {
			t.Fatalf("slices with same Elem should be equal")
		}
	})

	t.Run("Array compares by ArrayLen and Elem", func(t *testing.T) {
		t.Parallel()
		a := &node.TypeRef{TypeKind: node.TypeRefArray, ArrayLen: 3, Elem: namedRef("", "int")}
		b := &node.TypeRef{TypeKind: node.TypeRefArray, ArrayLen: 3, Elem: namedRef("", "int")}
		c := &node.TypeRef{TypeKind: node.TypeRefArray, ArrayLen: 4, Elem: namedRef("", "int")}
		if !a.Equal(b) {
			t.Fatalf("arrays with same ArrayLen and Elem should be equal")
		}
		if a.Equal(c) {
			t.Fatalf("arrays with different ArrayLen should not be equal")
		}
	})

	t.Run("Map compares by KeyType and ValueType", func(t *testing.T) {
		t.Parallel()
		a := &node.TypeRef{
			TypeKind: node.TypeRefMap,
			MapKey:   namedRef("", "string"),
			MapValue: namedRef("", "int"),
		}
		b := &node.TypeRef{
			TypeKind: node.TypeRefMap,
			MapKey:   namedRef("", "string"),
			MapValue: namedRef("", "int"),
		}
		c := &node.TypeRef{
			TypeKind: node.TypeRefMap,
			MapKey:   namedRef("", "string"),
			MapValue: namedRef("", "bool"),
		}
		if !a.Equal(b) {
			t.Fatalf("maps with same Key/Value should be equal")
		}
		if a.Equal(c) {
			t.Fatalf("maps with different Value should not be equal")
		}
	})

	t.Run("Func compares parameters and returns recursively", func(t *testing.T) {
		t.Parallel()
		a := &node.TypeRef{
			TypeKind:    node.TypeRefFunc,
			FuncParams:  []*node.TypeRef{namedRef("", "int")},
			FuncReturns: []*node.TypeRef{namedRef("", "string")},
		}
		b := &node.TypeRef{
			TypeKind:    node.TypeRefFunc,
			FuncParams:  []*node.TypeRef{namedRef("", "int")},
			FuncReturns: []*node.TypeRef{namedRef("", "string")},
		}
		c := &node.TypeRef{
			TypeKind:    node.TypeRefFunc,
			FuncParams:  []*node.TypeRef{namedRef("", "int")},
			FuncReturns: []*node.TypeRef{namedRef("", "bool")},
		}
		d := &node.TypeRef{
			TypeKind:    node.TypeRefFunc,
			FuncParams:  []*node.TypeRef{namedRef("", "int"), namedRef("", "int")},
			FuncReturns: []*node.TypeRef{namedRef("", "string")},
		}
		if !a.Equal(b) {
			t.Fatalf("funcs with same signature should be equal")
		}
		if a.Equal(c) {
			t.Fatalf("funcs with different return type should not be equal")
		}
		if a.Equal(d) {
			t.Fatalf("funcs with different param count should not be equal")
		}
	})

	t.Run("Named refs with same package and name and type args are equal", func(t *testing.T) {
		t.Parallel()
		a := namedRef("ctx", "Container")
		a.TypeArgs = []*node.TypeRef{namedRef("", "int")}
		b := namedRef("ctx", "Container")
		b.TypeArgs = []*node.TypeRef{namedRef("", "int")}
		if !a.Equal(b) {
			t.Fatalf("generic named refs with same args should be equal")
		}
	})

	t.Run("Named refs with different type args are not equal", func(t *testing.T) {
		t.Parallel()
		a := namedRef("ctx", "Container")
		a.TypeArgs = []*node.TypeRef{namedRef("", "int")}
		b := namedRef("ctx", "Container")
		b.TypeArgs = []*node.TypeRef{namedRef("", "string")}
		if a.Equal(b) {
			t.Fatalf("generic named refs with different args should not be equal")
		}
	})

	t.Run("TypeParam refs compare by Name", func(t *testing.T) {
		t.Parallel()
		a := &node.TypeRef{TypeKind: node.TypeRefTypeParam, Name: "T"}
		b := &node.TypeRef{TypeKind: node.TypeRefTypeParam, Name: "T"}
		c := &node.TypeRef{TypeKind: node.TypeRefTypeParam, Name: "U"}
		if !a.Equal(b) {
			t.Fatalf("TypeParam refs with same Name should be equal")
		}
		if a.Equal(c) {
			t.Fatalf("TypeParam refs with different Names should not be equal")
		}
	})

	t.Run("AnonStruct refs compare by inline fields and embeds", func(t *testing.T) {
		t.Parallel()
		a := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Fields: []*node.Field{
			{Name: "ID", Type: namedRef("", "string")},
		}}
		b := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Fields: []*node.Field{
			{Name: "ID", Type: namedRef("", "string")},
		}}
		c := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Fields: []*node.Field{
			{Name: "ID", Type: namedRef("", "int")},
		}}
		d := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Fields: []*node.Field{
			{Name: "ID", Type: namedRef("", "string"), Tag: `json:"id"`},
		}}
		e := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Fields: []*node.Field{
			{Name: "ID", Type: namedRef("", "string")},
			{Name: "Name", Type: namedRef("", "string")},
		}}
		f := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Embeds: []*node.Embed{{Type: namedRef("io", "Reader")}}}
		g := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Embeds: []*node.Embed{{Type: namedRef("io", "Reader")}}}
		h := &node.TypeRef{TypeKind: node.TypeRefAnonStruct, Embeds: []*node.Embed{{Type: namedRef("io", "Writer")}}}
		if !a.Equal(b) {
			t.Fatalf("anon structs with same fields should be equal")
		}
		if a.Equal(c) {
			t.Fatalf("anon structs with different field types should not be equal")
		}
		if a.Equal(d) {
			t.Fatalf("anon structs differing only in tag should not be equal")
		}
		if a.Equal(e) {
			t.Fatalf("anon structs with different field counts should not be equal")
		}
		if !f.Equal(g) {
			t.Fatalf("anon structs with same embeds should be equal")
		}
		if f.Equal(h) {
			t.Fatalf("anon structs with different embeds should not be equal")
		}
	})

	t.Run("AnonInterface refs compare by inline methods and embeds", func(t *testing.T) {
		t.Parallel()

		readByte := anonInterfaceWithRead(
			[]*node.Param{{Name: "p", Type: byteSliceRef()}},
			[]*node.TypeRef{namedRef("", "int"), namedRef("", "error")},
			false,
		)
		readByteCopy := anonInterfaceWithRead(
			[]*node.Param{{Name: "p", Type: byteSliceRef()}},
			[]*node.TypeRef{namedRef("", "int"), namedRef("", "error")},
			false,
		)
		readString := anonInterfaceWithRead(
			[]*node.Param{{Name: "p", Type: namedRef("", "string")}},
			[]*node.TypeRef{namedRef("", "int"), namedRef("", "error")},
			false,
		)
		writeByte := anonInterfaceWithMethod(
			"Write",
			[]*node.Param{{Name: "p", Type: byteSliceRef()}},
			[]*node.TypeRef{namedRef("", "int"), namedRef("", "error")},
			false,
		)
		readByteSingleReturn := anonInterfaceWithRead(
			[]*node.Param{{Name: "p", Type: byteSliceRef()}},
			[]*node.TypeRef{namedRef("", "int")},
			false,
		)
		readTwoParams := anonInterfaceWithRead(
			[]*node.Param{
				{Name: "p", Type: byteSliceRef()},
				{Name: "n", Type: namedRef("", "int")},
			},
			[]*node.TypeRef{namedRef("", "int"), namedRef("", "error")},
			false,
		)
		readVariadic := anonInterfaceWithRead(
			[]*node.Param{{Name: "p", Type: byteSliceRef()}},
			[]*node.TypeRef{namedRef("", "int"), namedRef("", "error")},
			true,
		)
		embedsReader := &node.TypeRef{
			TypeKind: node.TypeRefAnonInterface,
			Embeds:   []*node.Embed{{Type: namedRef("io", "Reader")}},
		}
		embedsReaderCopy := &node.TypeRef{
			TypeKind: node.TypeRefAnonInterface,
			Embeds:   []*node.Embed{{Type: namedRef("io", "Reader")}},
		}
		embedsWriter := &node.TypeRef{
			TypeKind: node.TypeRefAnonInterface,
			Embeds:   []*node.Embed{{Type: namedRef("io", "Writer")}},
		}
		empty := &node.TypeRef{TypeKind: node.TypeRefAnonInterface}

		cases := []struct {
			name      string
			a, b      *node.TypeRef
			wantEqual bool
		}{
			{"same method shape", readByte, readByteCopy, true},
			{"different param type", readByte, readString, false},
			{"different method name", readByte, writeByte, false},
			{"different return count", readByte, readByteSingleReturn, false},
			{"different param count", readByte, readTwoParams, false},
			{"variadic differs from non-variadic", readByte, readVariadic, false},
			{"same embeds", embedsReader, embedsReaderCopy, true},
			{"different embeds", embedsReader, embedsWriter, false},
			{"different method counts", empty, readByte, false},
			{"different embed counts", empty, embedsReader, false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				if got := tc.a.Equal(tc.b); got != tc.wantEqual {
					t.Fatalf("Equal = %v, want %v (a=%+v b=%+v)", got, tc.wantEqual, tc.a, tc.b)
				}
			})
		}
	})

	t.Run("unknown TypeRefKind returns false", func(t *testing.T) {
		t.Parallel()
		a := &node.TypeRef{TypeKind: node.TypeRefKind(99)}
		b := &node.TypeRef{TypeKind: node.TypeRefKind(99)}
		if a.Equal(b) {
			t.Fatalf("unknown TypeRefKind should compare unequal as a defensive default")
		}
	})
}
