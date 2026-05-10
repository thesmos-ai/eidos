// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

import "go.thesmos.sh/eidos/core/directive"

// Kind values for every concrete node type. Declared as
// [directive.Kind] so directive [Schema] AppliesTo lists reference
// them directly with full type safety:
//
//	directive.NewSchema("mock").On(node.KindInterface)
//
// The constants are the single source of truth for kind names —
// frontends populate [BaseNode.kind] (via the concrete type's Kind()
// method), the store indexes by these, and directive schemas
// constrain by them.
const (
	KindPackage     directive.Kind = "package"
	KindFile        directive.Kind = "file"
	KindImport      directive.Kind = "import"
	KindStruct      directive.Kind = "struct"
	KindInterface   directive.Kind = "interface"
	KindMethod      directive.Kind = "method"
	KindField       directive.Kind = "field"
	KindFunction    directive.Kind = "function"
	KindVariable    directive.Kind = "variable"
	KindConstant    directive.Kind = "constant"
	KindEnum        directive.Kind = "enum"
	KindEnumVariant directive.Kind = "enum_variant"
	KindAlias       directive.Kind = "alias"
	KindEmbed       directive.Kind = "embed"
	KindTypeRef     directive.Kind = "type_ref"
	KindParam       directive.Kind = "param"
	KindTypeParam   directive.Kind = "type_param"
)
