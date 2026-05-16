// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"unicode"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/emit/refconv"
	"go.thesmos.sh/eidos/node"
)

// emitBuilder builds the `<Name>Builder` struct + its
// `With<Field>` setter methods + the `Build() *<Name>` finalizer
// for one source struct. The struct lands in pkg under the
// configured `<src.Name><Suffix>` name; PackageBuilder.Anchor
// has already stamped src as the default Origin so every decl
// back-links automatically.
func (p *Plugin) emitBuilder(pkg *builder.PackageBuilder, src *node.Struct) {
	builderName := src.Name + p.opts.Suffix
	fields := exportedFields(src)
	srcRef := emit.External(src.Package, src.Name)

	pkg.Struct(builderName, func(b *builder.StructBuilder) {
		b.Docs(
			builderName + " accumulates field values for " + src.Name +
				" and produces the populated value via " + builderName + ".Build.",
		)
		for _, f := range fields {
			b.Field(f.Name, refconv.FromNode(f.Type), nil)
		}

		recv := emit.Ptr(emit.Internal(b.Node()))
		for _, f := range fields {
			fieldName := f.Name
			fieldType := refconv.FromNode(f.Type)
			b.Method(p.setterName(fieldName), func(m *builder.MethodBuilder) {
				m.Receiver("b", recv)
				m.Param("value", fieldType)
				m.Return(recv)
				m.Body(
					emit.NewAssign(
						[]*emit.Expr{emit.NewField(emit.NewIdent("b"), fieldName)},
						"=",
						[]*emit.Expr{emit.NewIdent("value")},
					),
					emit.NewReturn(emit.NewIdent("b")),
				)
			})
		}

		b.Method("Build", func(m *builder.MethodBuilder) {
			m.Receiver("b", recv)
			m.Return(emit.Ptr(srcRef))
			m.Body(buildFinalizerBody(src, fields)...)
		})
	})
}

// buildFinalizerBody returns the statement list for the
// `Build()` method: `return &<Source>{Field: b.Field, ...}` —
// projects each accumulated value back into a freshly-constructed
// source struct.
func buildFinalizerBody(src *node.Struct, fields []*node.Field) []*emit.Stmt {
	srcRef := emit.External(src.Package, src.Name)
	keys := make([]string, 0, len(fields))
	values := make([]*emit.Expr, 0, len(fields))
	for _, f := range fields {
		keys = append(keys, f.Name)
		values = append(values, emit.NewField(emit.NewIdent("b"), f.Name))
	}
	composite := emit.NewCompositeKeyed(srcRef, keys, values)
	return []*emit.Stmt{
		emit.NewReturn(emit.NewUnary("&", composite)),
	}
}

// exportedFields returns the subset of src.Fields whose Name
// begins with an uppercase letter — the Go convention for
// package-exported identifiers. Builders only project exported
// fields; consumers cannot set unexported ones from outside the
// declaring package anyway.
func exportedFields(src *node.Struct) []*node.Field {
	out := make([]*node.Field, 0, len(src.Fields))
	for _, f := range src.Fields {
		if isExported(f.Name) {
			out = append(out, f)
		}
	}
	return out
}

// isExported reports whether name begins with an uppercase
// letter — the Go convention for package-exported identifiers.
func isExported(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}
