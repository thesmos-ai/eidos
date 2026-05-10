// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Kind values for every concrete emit type. Declared as
// [directive.Kind] so directive schemas reference them directly:
//
//	directive.NewSchema("repo").On(emit.KindStruct)
//
// The set parallels the [node] package's kinds and adds emit-only
// kinds for [Ref] variants, [Stmt], and [Expr]. Frontends never
// produce these — they appear only in the generator-built emit tree.
const (
	// Declaration kinds parallel to node.
	KindPackage     directive.Kind = "emit.package"
	KindFile        directive.Kind = "emit.file"
	KindImport      directive.Kind = "emit.import"
	KindStruct      directive.Kind = "emit.struct"
	KindInterface   directive.Kind = "emit.interface"
	KindMethod      directive.Kind = "emit.method"
	KindField       directive.Kind = "emit.field"
	KindFunction    directive.Kind = "emit.function"
	KindVariable    directive.Kind = "emit.variable"
	KindConstant    directive.Kind = "emit.constant"
	KindEnum        directive.Kind = "emit.enum"
	KindEnumVariant directive.Kind = "emit.enum_variant"
	KindAlias       directive.Kind = "emit.alias"
	KindEmbed       directive.Kind = "emit.embed"
	KindParam       directive.Kind = "emit.param"
	KindTypeParam   directive.Kind = "emit.type_param"

	// Reference kinds.
	KindTypeRef      directive.Kind = "emit.type_ref"
	KindExternalRef  directive.Kind = "emit.external_ref"
	KindBuiltinRef   directive.Kind = "emit.builtin_ref"
	KindCompositeRef directive.Kind = "emit.composite_ref"

	// Body kinds.
	KindStmt directive.Kind = "emit.stmt"
	KindExpr directive.Kind = "emit.expr"

	// Composition primitives.
	KindSlot directive.Kind = "emit.slot"
)
