// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import "go.thesmos.sh/eidos/core/directive"

// Function is a standalone (non-method) function emit. Cross-cutting
// generators inject pre- and post-body statements through the
// "prebody" and "postbody" slots — same convention as [Method].
type Function struct {
	BaseEmit

	// Name is the function identifier.
	Name string

	// Package is the package name the rendered file declares.
	Package string

	// Params are the function's positional parameters in source
	// order.
	Params []*Param

	// Returns are the function's return types in source order.
	Returns []Ref

	// TypeParams are the function's generic type parameters.
	TypeParams []*TypeParam

	// Body holds the function's statement body in source order.
	Body []*Stmt

	// Target identifies where the backend writes this function's
	// rendered output.
	Target Target

	slotMap
}

// Kind returns [KindFunction].
func (*Function) Kind() directive.Kind { return KindFunction }

// QName returns the qualified name "Package.Name", or just "Name"
// when Package is empty.
func (f *Function) QName() string {
	if f.Package == "" {
		return f.Name
	}
	return f.Package + "." + f.Name
}

// Prebody returns the "prebody" slot for cross-cutting contributions
// that run before [Function.Body].
func (f *Function) Prebody() *Slot { return f.slot(f, "prebody", KindStmt) }

// Postbody returns the "postbody" slot for cross-cutting
// contributions that run after [Function.Body] returns.
func (f *Function) Postbody() *Slot { return f.slot(f, "postbody", KindStmt) }

// ParamsSlot returns the "params" slot for cross-cutting parameter
// injection.
func (f *Function) ParamsSlot() *Slot { return f.slot(f, "params", KindParam) }

// ReturnsSlot returns the "returns" slot for cross-cutting
// return-type injection.
func (f *Function) ReturnsSlot() *Slot { return f.slot(f, "returns", "") }

// Slot returns the named slot, creating it lazily.
func (f *Function) Slot(name string) *Slot { return f.slot(f, name, "") }

// IsVariadic reports whether the function's last positional
// parameter is variadic.
func (f *Function) IsVariadic() bool {
	n := len(f.Params)
	return n > 0 && f.Params[n-1].Variadic
}

// IsGeneric reports whether the function declares generic type
// parameters.
func (f *Function) IsGeneric() bool { return len(f.TypeParams) > 0 }

// ParamCount returns the number of declared positional parameters.
func (f *Function) ParamCount() int { return len(f.Params) }

// ReturnCount returns the number of declared return values.
func (f *Function) ReturnCount() int { return len(f.Returns) }

// ParamByName returns the parameter with the given name, or nil
// when no such parameter exists. Empty name returns nil.
func (f *Function) ParamByName(name string) *Param {
	if name == "" {
		return nil
	}
	for _, p := range f.Params {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// ParamAt returns the parameter at the given index, or nil when out
// of range.
func (f *Function) ParamAt(i int) *Param {
	if i < 0 || i >= len(f.Params) {
		return nil
	}
	return f.Params[i]
}

// ReturnAt returns the return type at the given index, or nil when
// out of range.
func (f *Function) ReturnAt(i int) Ref {
	if i < 0 || i >= len(f.Returns) {
		return nil
	}
	return f.Returns[i]
}
