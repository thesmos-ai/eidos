// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// Language is the language identifier the Go adapter answers
// to. Plugins dispatch [plugin.Plugin.Outputs] /
// [plugin.Plugin.Templates] / [plugin.TemplateProvider.TemplateFuncs]
// on this constant to route per-language behaviour to the Go
// side.
const Language = "golang"

// IsExported reports whether name follows Go's
// exported-identifier rule — first rune upper-case ASCII for
// the simple cases plugins target. Empty input is unexported
// by definition.
func IsExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}

// ExportedFields returns the exported fields of s in source
// order. Plugin templates iterate the result to render
// setters / projections that mirror what user code can
// reach.
func ExportedFields(s *node.Struct) []*node.Field {
	out := make([]*node.Field, 0, len(s.Fields))
	for _, f := range s.Fields {
		if IsExported(f.Name) {
			out = append(out, f)
		}
	}
	return out
}

// IsSlice reports whether t is a non-byte slice. `[]byte`
// routes through [IsByteSlice] instead so plugin templates
// can pick a bytes-flavoured setter pair for that special
// case without seeing it twice.
func IsSlice(t *node.TypeRef) bool {
	return t != nil && t.IsSlice() && !IsByteSlice(t)
}

// IsMap reports whether t is a map type.
func IsMap(t *node.TypeRef) bool {
	return t != nil && t.IsMap()
}

// IsByteSlice reports whether t is `[]byte` — the only slice
// shape Go renders idiomatically through a bytes-string
// convenience setter pair rather than the variadic /
// append pair every other slice uses.
func IsByteSlice(t *node.TypeRef) bool {
	if t == nil || !t.IsSlice() || t.Elem == nil {
		return false
	}
	return t.Elem.IsBuiltin() && t.Elem.Name == "byte"
}

// FieldType returns the [emit.Ref] for a field's declared
// type — the shape plugin templates feed to the backend's
// `renderType` funcmap for setter parameters, internal
// value-field types, and composite-literal types.
func FieldType(f *node.Field) emit.Ref {
	return FromNode(f.Type)
}

// ElemType returns the [emit.Ref] for the slice / array
// element type of t.
func ElemType(t *node.TypeRef) emit.Ref {
	return FromNode(t.Elem)
}

// MapKeyType returns the [emit.Ref] for the map-key type of
// t.
func MapKeyType(t *node.TypeRef) emit.Ref {
	return FromNode(t.MapKey)
}

// MapValType returns the [emit.Ref] for the map-value type
// of t.
func MapValType(t *node.TypeRef) emit.Ref {
	return FromNode(t.MapValue)
}

// TypeParams lifts s's source-side type-parameter list into
// the [emit.TypeParam] form the backend's `renderTypeParams`
// funcmap entry consumes. Constraint conversion runs through
// [ConstraintFromNode] so external constraint types register
// their import on the rendered file automatically. Returns
// nil for non-generic structs so the template's
// `renderTypeParams` call emits no bracket list.
func TypeParams(s *node.Struct) []*emit.TypeParam {
	if len(s.TypeParams) == 0 {
		return nil
	}
	out := make([]*emit.TypeParam, len(s.TypeParams))
	for i, p := range s.TypeParams {
		out[i] = &emit.TypeParam{
			Name:       p.Name,
			Constraint: ConstraintFromNode(p.Constraint),
		}
	}
	return out
}

// TypeArgs returns the bracketed parameter-name use form for
// s's type-parameter list (`[T, K]`) or the empty string for
// non-generic structs. Plugin templates use the result for
// receiver / return / composite-literal type-arg threading
// on each rendered method.
func TypeArgs(s *node.Struct) string {
	if len(s.TypeParams) == 0 {
		return ""
	}
	names := make([]string, len(s.TypeParams))
	for i, p := range s.TypeParams {
		names[i] = p.Name
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// SelfType returns the [emit.Ref] for s's own type
// instantiation — `Container` for non-generic structs,
// `Container[T]` for generics. Plugin templates use the
// result for emitted helper types' internal value fields,
// for `Build`-style return types, and for `From`-style
// parameter types. Same-package elision in the Go backend's
// `renderType` drops the package qualifier when the
// rendered file lives in the source package.
func SelfType(s *node.Struct) emit.Ref {
	if len(s.TypeParams) == 0 {
		return emit.External(s.Package, s.Name)
	}
	args := make([]emit.Ref, len(s.TypeParams))
	for i, p := range s.TypeParams {
		args[i] = emit.Builtin(p.Name)
	}
	return emit.External(s.Package, s.Name, args...)
}

// FuncMap returns the conventional template-side funcmap the
// shared Go helpers register under. Plugins compose this
// with their plugin-specific entries in their own
// [plugin.TemplateProvider.TemplateFuncs] implementation —
// the bundled keys cover the predicates / lifters every Go
// template needs so the plugin only declares its own
// extensions.
//
// Keys:
//
//	isExported     → IsExported
//	exportedFields → ExportedFields
//	isSlice        → IsSlice
//	isMap          → IsMap
//	isByteSlice    → IsByteSlice
//	fieldType      → FieldType
//	elemType       → ElemType
//	mapKeyType     → MapKeyType
//	mapValType     → MapValType
//	typeParams     → TypeParams
//	typeArgs       → TypeArgs
//	selfType       → SelfType
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"isExported":     IsExported,
		"exportedFields": ExportedFields,
		"isSlice":        IsSlice,
		"isMap":          IsMap,
		"isByteSlice":    IsByteSlice,
		"fieldType":      FieldType,
		"elemType":       ElemType,
		"mapKeyType":     MapKeyType,
		"mapValType":     MapValType,
		"typeParams":     TypeParams,
		"typeArgs":       TypeArgs,
		"selfType":       SelfType,
	}
}
