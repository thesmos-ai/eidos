// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"fmt"

	"go.thesmos.sh/eidos/emit"
)

// AppendField appends f to host's "fields" slot, stamping
// [emit.Provenance.SetBy] with the Context's plugin identifier.
// Optional id is recorded in [emit.Provenance.ID] for later
// positional inserts via [emit.Slot.InsertBefore] /
// [emit.Slot.InsertAfter].
//
// host must be a *[emit.Struct]; other kinds return
// [ErrUnsupportedHost]. The wrapped slot append surfaces
// [emit.ErrSlotElementType] when f.Kind() doesn't match the slot's
// declared element kind.
func (c *Context) AppendField(host emit.Node, f *emit.Field, id ...string) error {
	if err := nilCheck(host, "field"); err != nil {
		return err
	}
	owner, ok := host.(*emit.Struct)
	if !ok {
		return fmt.Errorf("%w: AppendField expects *emit.Struct, got %T", ErrUnsupportedHost, host)
	}
	f.Owner = owner
	return owner.FieldsSlot().Append(f, c.Provenance(id...))
}

// AppendMethod appends m to host's "methods" slot, stamping
// provenance with the Context's plugin identifier.
//
// host may be a *[emit.Struct], *[emit.Interface], or *[emit.Alias];
// other kinds return [ErrUnsupportedHost].
func (c *Context) AppendMethod(host emit.Node, m *emit.Method, id ...string) error {
	if err := nilCheck(host, "method"); err != nil {
		return err
	}
	switch owner := host.(type) {
	case *emit.Struct:
		m.Owner = owner
		return owner.MethodsSlot().Append(m, c.Provenance(id...))
	case *emit.Interface:
		m.Owner = owner
		return owner.MethodsSlot().Append(m, c.Provenance(id...))
	case *emit.Alias:
		m.Owner = owner
		return owner.MethodsSlot().Append(m, c.Provenance(id...))
	default:
		return fmt.Errorf(
			"%w: AppendMethod expects *emit.Struct, *emit.Interface, or *emit.Alias, got %T",
			ErrUnsupportedHost, host,
		)
	}
}

// AppendEmbed appends e to host's "embeds" slot. host may be a
// *[emit.Struct] or *[emit.Interface].
func (c *Context) AppendEmbed(host emit.Node, e *emit.Embed, id ...string) error {
	if err := nilCheck(host, "embed"); err != nil {
		return err
	}
	switch owner := host.(type) {
	case *emit.Struct:
		e.Owner = owner
		return owner.EmbedsSlot().Append(e, c.Provenance(id...))
	case *emit.Interface:
		e.Owner = owner
		return owner.EmbedsSlot().Append(e, c.Provenance(id...))
	default:
		return fmt.Errorf("%w: AppendEmbed expects *emit.Struct or *emit.Interface, got %T", ErrUnsupportedHost, host)
	}
}

// AppendVariant appends v to host's "variants" slot. host must be
// a *[emit.Enum].
func (c *Context) AppendVariant(host *emit.Enum, v *emit.EnumVariant, id ...string) error {
	if host == nil {
		return fmt.Errorf("%w: variant host is nil", ErrNilHost)
	}
	v.Owner = host
	return host.VariantsSlot().Append(v, c.Provenance(id...))
}

// AppendParam appends p to host's "params" slot. host may be a
// *[emit.Function] or *[emit.Method].
func (c *Context) AppendParam(host emit.Node, p *emit.Param, id ...string) error {
	if err := nilCheck(host, "param"); err != nil {
		return err
	}
	switch owner := host.(type) {
	case *emit.Function:
		p.Owner = owner
		return owner.ParamsSlot().Append(p, c.Provenance(id...))
	case *emit.Method:
		p.Owner = owner
		return owner.ParamsSlot().Append(p, c.Provenance(id...))
	default:
		return fmt.Errorf("%w: AppendParam expects *emit.Function or *emit.Method, got %T", ErrUnsupportedHost, host)
	}
}

// AppendPrebody appends stmt to host's "prebody" slot — code that
// runs before the host's typed Body. host may be a *[emit.Function]
// or *[emit.Method].
func (c *Context) AppendPrebody(host emit.Node, stmt *emit.Stmt, id ...string) error {
	if err := nilCheck(host, "prebody"); err != nil {
		return err
	}
	switch owner := host.(type) {
	case *emit.Function:
		return owner.Prebody().Append(stmt, c.Provenance(id...))
	case *emit.Method:
		return owner.Prebody().Append(stmt, c.Provenance(id...))
	default:
		return fmt.Errorf("%w: AppendPrebody expects *emit.Function or *emit.Method, got %T", ErrUnsupportedHost, host)
	}
}

// AppendPostbody appends stmt to host's "postbody" slot — code that
// runs after the host's typed Body returns.
func (c *Context) AppendPostbody(host emit.Node, stmt *emit.Stmt, id ...string) error {
	if err := nilCheck(host, "postbody"); err != nil {
		return err
	}
	switch owner := host.(type) {
	case *emit.Function:
		return owner.Postbody().Append(stmt, c.Provenance(id...))
	case *emit.Method:
		return owner.Postbody().Append(stmt, c.Provenance(id...))
	default:
		return fmt.Errorf("%w: AppendPostbody expects *emit.Function or *emit.Method, got %T", ErrUnsupportedHost, host)
	}
}

// AppendTag appends tag to host's "tags" slot — cross-cutting
// struct-tag entries on a single field. host must be a *[emit.Field].
//
// The tag arg is an [emit.Tag] (key + value); the renderer renders
// it as `key:"escaped value"` and joins it with the field's
// directly-declared [emit.Field.Tag].
func (c *Context) AppendTag(host *emit.Field, tag *emit.Tag, id ...string) error {
	if host == nil {
		return fmt.Errorf("%w: tag host is nil", ErrNilHost)
	}
	return host.Tags().Append(tag, c.Provenance(id...))
}

// AppendTop appends decl to file.Top() — the file-level slot
// rendered above free-floating declarations. decl is any [emit.Node]
// the file's top slot accepts (Struct, Interface, Function, etc.).
func (c *Context) AppendTop(file *emit.File, decl emit.Node, id ...string) error {
	if file == nil {
		return fmt.Errorf("%w: file is nil", ErrNilHost)
	}
	return file.Top().Append(decl, c.Provenance(id...))
}

// AppendBottom appends decl to file.Bottom() — the file-level slot
// rendered below free-floating declarations.
func (c *Context) AppendBottom(file *emit.File, decl emit.Node, id ...string) error {
	if file == nil {
		return fmt.Errorf("%w: file is nil", ErrNilHost)
	}
	return file.Bottom().Append(decl, c.Provenance(id...))
}

// AppendInit appends stmt to file.Init() — composed into the file's
// `func init()` body. Multiple plugins' contributions concatenate in
// capability-topological order.
func (c *Context) AppendInit(file *emit.File, stmt *emit.Stmt, id ...string) error {
	if file == nil {
		return fmt.Errorf("%w: file is nil", ErrNilHost)
	}
	return file.Init().Append(stmt, c.Provenance(id...))
}

// AppendImport appends an explicit [emit.Import] to file.ImportsSlot()
// — the canonical path for cross-cutting plugins that need a file
// to carry an import the typed Imports list doesn't already declare
// (typically side-effect imports referenced by code the plugin
// contributes elsewhere).
func (c *Context) AppendImport(file *emit.File, imp *emit.Import, id ...string) error {
	if file == nil {
		return fmt.Errorf("%w: file is nil", ErrNilHost)
	}
	imp.Owner = file
	return file.ImportsSlot().Append(imp, c.Provenance(id...))
}
