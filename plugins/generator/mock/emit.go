// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mock

import (
	"strconv"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/emit/refconv"
	"go.thesmos.sh/eidos/node"
)

// emitMock builds the mock struct + its dispatch methods for one
// source interface into pkg. The struct lands under the
// `<iface.Name><Suffix>` name; every emit decl back-links to
// iface as its origin so the Layout phase routes the output
// alongside the source.
func (p *Plugin) emitMock(pkg *builder.PackageBuilder, iface *node.Interface) {
	mockName := iface.Name + p.opts.Suffix
	methods := mockableMethods(iface)

	pkg.Struct(mockName, func(s *builder.StructBuilder) {
		s.Origin(iface)
		s.Docs(mockName + " is a func-valued mock implementation of " + iface.Name + ".")
		MetaIface.Set(s.Node().Meta(), iface.QName(), Name)

		for _, m := range methods {
			s.Field(p.fieldNameFor(m.Name), funcRefFor(m), nil)
		}

		recv := emit.Ptr(emit.Internal(s.Node()))
		for _, m := range methods {
			field := p.fieldNameFor(m.Name)
			s.Method(m.Name, func(mb *builder.MethodBuilder) {
				mb.Receiver("m", recv)
				for i, prm := range m.Params {
					mb.Param(paramName(prm.Name, i), refconv.FromNode(prm.Type))
				}
				for i, ret := range m.Returns {
					mb.Return(refconv.FromNode(ret), returnName(i))
				}
				mb.Body(p.dispatchBody(m)...)
				MetaField.Set(mb.Node().Meta(), field, Name)
			})
		}
	})
}

// dispatchBody returns the statement list for one mock method's
// body — a nil-check on the override field, plus a trailing naked
// return only when the method has named returns to yield zero
// values for.
func (p *Plugin) dispatchBody(m *node.Method) []*emit.Stmt {
	field := p.fieldNameFor(m.Name)
	args := make([]*emit.Expr, 0, len(m.Params))
	for i, prm := range m.Params {
		args = append(args, emit.NewIdent(paramName(prm.Name, i)))
	}
	call := emit.NewCall(emit.NewField(emit.NewIdent("m"), field), args...)

	var dispatch *emit.Stmt
	if len(m.Returns) == 0 {
		dispatch = emit.NewExprStmt(call)
	} else {
		dispatch = emit.NewReturn(call)
	}
	cond := emit.NewBinary(
		emit.NewField(emit.NewIdent("m"), field),
		"!=",
		emit.NewIdent("nil"),
	)
	stmts := []*emit.Stmt{emit.NewIf(cond, []*emit.Stmt{dispatch})}
	if len(m.Returns) > 0 {
		stmts = append(stmts, emit.NewReturn())
	}
	return stmts
}

// mockableMethods returns iface's methods minus any with a negated
// `+gen:mock` directive on the method itself, so callers can opt
// individual methods out of mock dispatch.
func mockableMethods(iface *node.Interface) []*node.Method {
	out := make([]*node.Method, 0, len(iface.Methods))
	for _, m := range iface.Methods {
		if m.HasNegatedDirective(DirectiveName) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// funcRefFor returns the `func(<params>) (<returns>)` type for the
// field that backs a mock method. Field-position types omit
// parameter names, so this synthesises the bare type ref.
func funcRefFor(m *node.Method) emit.Ref {
	params := make([]emit.Ref, 0, len(m.Params))
	for _, p := range m.Params {
		params = append(params, refconv.FromNode(p.Type))
	}
	returns := make([]emit.Ref, 0, len(m.Returns))
	for _, r := range m.Returns {
		returns = append(returns, refconv.FromNode(r))
	}
	return emit.FuncOf(params, returns)
}

// paramName returns the in-method identifier for parameter index
// i — the source name when non-empty, or `arg<N>` when the source
// omitted the name (anonymous param in an interface signature,
// common in Go).
func paramName(name string, i int) string {
	if name != "" {
		return name
	}
	return "arg" + strconv.Itoa(i)
}

// returnName returns the named-return identifier for return index
// i. The mock always uses positional `_v<N>` names so the naked
// `return` in [Plugin.dispatchBody] yields the zero value of each
// return type.
func returnName(i int) string { return "_v" + strconv.Itoa(i) }
