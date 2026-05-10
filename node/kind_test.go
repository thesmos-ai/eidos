// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
)

// TestKind asserts that every concrete node kind reports the
// expected directive.Kind constant. Each Kind() method lives on its
// own source file but the test sits here so adding a new kind is one
// table-row change rather than a one-test-per-file ritual.
func TestKind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		n    node.Node
		want directive.Kind
	}{
		{"Package", &node.Package{}, node.KindPackage},
		{"File", &node.File{}, node.KindFile},
		{"Import", &node.Import{}, node.KindImport},
		{"Struct", &node.Struct{}, node.KindStruct},
		{"Interface", &node.Interface{}, node.KindInterface},
		{"Method", &node.Method{}, node.KindMethod},
		{"Field", &node.Field{}, node.KindField},
		{"Function", &node.Function{}, node.KindFunction},
		{"Variable", &node.Variable{}, node.KindVariable},
		{"Constant", &node.Constant{}, node.KindConstant},
		{"Enum", &node.Enum{}, node.KindEnum},
		{"EnumVariant", &node.EnumVariant{}, node.KindEnumVariant},
		{"Alias", &node.Alias{}, node.KindAlias},
		{"Embed", &node.Embed{}, node.KindEmbed},
		{"TypeRef", &node.TypeRef{}, node.KindTypeRef},
		{"Param", &node.Param{}, node.KindParam},
		{"TypeParam", &node.TypeParam{}, node.KindTypeParam},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.n.Kind(); got != tc.want {
				t.Fatalf("Kind = %q, want %q", got, tc.want)
			}
		})
	}
}
