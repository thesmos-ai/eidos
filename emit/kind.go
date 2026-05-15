// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/kind"
)

// Kind values for every concrete emit type. Declared as
// [kind.Kind] so directive schemas reference them directly:
//
//	directive.NewSchema("repo").On(emit.KindStruct)
//
// The set parallels the [node] package's kinds and adds emit-only
// kinds for [Ref] variants, [Stmt], and [Expr]. Frontends never
// produce these — they appear only in the generator-built emit tree.
const (
	// Declaration kinds parallel to node.
	KindPackage     kind.Kind = "emit.package"
	KindFile        kind.Kind = "emit.file"
	KindImport      kind.Kind = "emit.import"
	KindStruct      kind.Kind = "emit.struct"
	KindInterface   kind.Kind = "emit.interface"
	KindMethod      kind.Kind = "emit.method"
	KindField       kind.Kind = "emit.field"
	KindFunction    kind.Kind = "emit.function"
	KindVariable    kind.Kind = "emit.variable"
	KindConstant    kind.Kind = "emit.constant"
	KindEnum        kind.Kind = "emit.enum"
	KindEnumVariant kind.Kind = "emit.enum_variant"
	KindAlias       kind.Kind = "emit.alias"
	KindEmbed       kind.Kind = "emit.embed"
	KindParam       kind.Kind = "emit.param"
	KindReturn      kind.Kind = "emit.return"
	KindTypeParam   kind.Kind = "emit.type_param"
	KindTag         kind.Kind = "emit.tag"

	// Reference kinds.
	KindTypeRef      kind.Kind = "emit.type_ref"
	KindExternalRef  kind.Kind = "emit.external_ref"
	KindBuiltinRef   kind.Kind = "emit.builtin_ref"
	KindCompositeRef kind.Kind = "emit.composite_ref"

	// Body kinds.
	KindStmt kind.Kind = "emit.stmt"
	KindExpr kind.Kind = "emit.expr"

	// Composition primitives.
	KindSlot kind.Kind = "emit.slot"
)
