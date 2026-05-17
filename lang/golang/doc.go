// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package golang holds the Go-language conventions every
// plugin generating Go output shares. It is the Go side of
// the per-language adapter pattern eidos plugins use to keep
// their cores language-agnostic.
//
// # Surface
//
// The package surface splits into three groups:
//
//   - Predicates ([IsExported], [IsByteSlice], [IsSlice],
//     [IsMap]) — classify source [node.TypeRef] / [node.Field]
//     shapes against Go's grammar so plugin templates can
//     branch correctly without re-implementing the rules.
//   - Lifters ([FromNode], [ConstraintFromNode], [FieldType],
//     [ElemType], [MapKeyType], [MapValType], [TypeParams],
//     [TypeArgs], [SelfType]) — convert source-side
//     [node.*] values into the [emit.Ref] / [emit.TypeParam]
//     forms the backend's `renderType` / `renderTypeParams`
//     funcmap entries consume.
//   - The [FuncMap] bundler — returns every entry above
//     keyed under the conventional template-side name
//     ("isExported", "selfType", "typeArgs", …). Plugins
//     compose it with their plugin-specific extensions in
//     their own [plugin.TemplateProvider.TemplateFuncs]
//     implementation.
//
// # Boundary
//
// The package depends only on [node] and [emit] — it neither
// reaches into a frontend (which would couple plugins to a
// source-language choice) nor into the Go backend (which
// would violate the depguard rule banning `plugins/` from
// importing `backend/`). Future per-language packages
// (`lang/rust`, `lang/typescript`) mirror this shape with
// language-flavoured equivalents.
package golang
