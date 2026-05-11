// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Method is a function attached to a [Struct] or [Interface]. A
// Method on an Interface has a nil [Method.Receiver]; a Method on a
// Struct carries the receiver type explicitly along with the
// receiver variable name.
//
// Method exposes four standard slots:
//
//   - "prebody"  — Stmts run before the method's Body.
//     Common target of cross-cutters (logging, validation, audit).
//   - "postbody" — Stmts run after the method's Body returns. Less
//     common but useful for deferred work.
//   - "params"   — Param injection (rare; usually parameters live
//     in the typed Params slice).
//   - "returns"  — Return-type injection (rare).
type Method struct {
	BaseEmit

	// Name is the method identifier.
	Name string

	// Receiver is the receiver type of a struct method. nil for
	// methods declared inside an interface.
	Receiver Ref

	// ReceiverName is the receiver variable name for the generated
	// method ("r" in `func (r *Repo) Get()`). Empty for interface
	// methods and for receiver declarations using the blank
	// receiver form.
	ReceiverName string

	// Params are the method's positional parameters in source order.
	Params []*Param

	// Returns are the method's return types in source order.
	Returns []Ref

	// TypeParams are the method's generic type parameters.
	TypeParams []*TypeParam

	// Body holds the method's statement body in source order.
	// Cross-cutting injection goes through the prebody / postbody
	// slots rather than mutating Body directly.
	Body []*Stmt

	// Owner is the [Struct] or [Interface] that declares this
	// method.
	//
	// Owner is excluded from JSON encoding to break the host →
	// child cycle. Deserialized graphs re-wire Owner via
	// [RewireOwners].
	Owner Node `json:"-"`

	slotMap
}

// Kind returns [KindMethod].
func (*Method) Kind() directive.Kind { return KindMethod }

// Prebody returns the "prebody" slot for cross-cutting contributions
// that run before [Method.Body].
func (m *Method) Prebody() *Slot { return m.slot(m, "prebody", KindStmt) }

// Postbody returns the "postbody" slot for cross-cutting
// contributions that run after [Method.Body] returns.
func (m *Method) Postbody() *Slot { return m.slot(m, "postbody", KindStmt) }

// ParamsSlot returns the "params" slot for cross-cutting parameter
// injection. Distinct from [Method.Params] (the typed field for the
// owning generator's direct content).
func (m *Method) ParamsSlot() *Slot { return m.slot(m, "params", KindParam) }

// ReturnsSlot returns the "returns" slot for cross-cutting
// return-type injection.
func (m *Method) ReturnsSlot() *Slot { return m.slot(m, "returns", "") }

// Slot returns the named slot, creating it lazily without an
// element-kind constraint. Used for custom slot names a plugin
// declares alongside its emit kinds.
func (m *Method) Slot(name string) *Slot { return m.slot(m, name, "") }

// HasReceiver reports whether the method has an explicit receiver.
func (m *Method) HasReceiver() bool { return m.Receiver != nil }

// IsVariadic reports whether the method's last positional parameter
// is variadic.
func (m *Method) IsVariadic() bool {
	n := len(m.Params)
	return n > 0 && m.Params[n-1].Variadic
}

// IsGeneric reports whether the method declares generic type
// parameters of its own.
func (m *Method) IsGeneric() bool { return len(m.TypeParams) > 0 }

// ParamCount returns the number of declared positional parameters.
func (m *Method) ParamCount() int { return len(m.Params) }

// ReturnCount returns the number of declared return values.
func (m *Method) ReturnCount() int { return len(m.Returns) }

// ParamByName returns the parameter with the given name, or nil
// when no such parameter exists. Empty name returns nil.
func (m *Method) ParamByName(name string) *Param {
	if name == "" {
		return nil
	}
	for _, p := range m.Params {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// ParamAt returns the parameter at the given index, or nil when
// out of range. Bounds-checked positional access.
func (m *Method) ParamAt(i int) *Param {
	if i < 0 || i >= len(m.Params) {
		return nil
	}
	return m.Params[i]
}

// ReturnAt returns the return type at the given index, or nil when
// out of range.
func (m *Method) ReturnAt(i int) Ref {
	if i < 0 || i >= len(m.Returns) {
		return nil
	}
	return m.Returns[i]
}
