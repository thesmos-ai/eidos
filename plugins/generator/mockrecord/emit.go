// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mockrecord

import (
	"strconv"
	"unicode"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// emitRecording wires the per-mock recording surface: a typed
// per-method call struct, a slice field on the mock for each
// method, and a Prebody contribution that appends to that slice
// at every dispatch. Returns the first error from a slot append —
// the slot API rejects nil entries, so a programmer error in this
// package would surface here rather than silently dropping.
func (p *Plugin) emitRecording(pkg *builder.PackageBuilder, mockStruct *emit.Struct) error {
	for _, m := range mockStruct.Methods {
		callStructName := mockStruct.Name + m.Name + p.opts.CallStructSuffix
		fieldName := m.Name + p.opts.FieldSuffix
		callStruct := emitCallStruct(pkg, callStructName, m)
		if err := appendCallsField(mockStruct, fieldName, callStruct); err != nil {
			return err
		}
		if err := appendRecordingPrebody(m, fieldName, callStruct); err != nil {
			return err
		}
	}
	return nil
}

// emitCallStruct builds the per-method `<MockName><Method>Call`
// struct that records one invocation's parameter values. Field
// names are exported derivations of the source parameter names
// (anonymous params land as `Arg<N>`). Returns the constructed
// struct for use as the element type of the slice field appended
// to the mock.
func emitCallStruct(
	pkg *builder.PackageBuilder,
	name string,
	m *emit.Method,
) *emit.Struct {
	var built *emit.Struct
	pkg.Struct(name, func(s *builder.StructBuilder) {
		s.Docs(name + " records one call to the matching mock method.")
		for i, p := range m.Params {
			s.Field(paramFieldName(p.Name, i), p.Type, nil)
		}
		built = s.Node()
	})
	return built
}

// appendCallsField appends a `<Method>Calls []<...>Call` field to
// the mock struct's [emit.Struct.FieldsSlot]. The slot is the
// supported cross-cutting extension surface — the field renders
// inline with the typed Fields the mock owns.
func appendCallsField(mockStruct *emit.Struct, name string, callStruct *emit.Struct) error {
	field := &emit.Field{
		Name:  name,
		Type:  emit.SliceOf(emit.Internal(callStruct)),
		Owner: mockStruct,
	}
	return mockStruct.FieldsSlot().Append(field, emit.Provenance{SetBy: Name})
}

// appendRecordingPrebody prepends one `m.<Method>Calls =
// append(m.<Method>Calls, <CallStruct>{...})` statement to the
// dispatch method's Prebody. The statement runs before the
// mock's nil-check, so a recorded call is captured even when no
// override is installed.
func appendRecordingPrebody(m *emit.Method, fieldName string, callStruct *emit.Struct) error {
	target := emit.NewField(emit.NewIdent("m"), fieldName)
	keys := make([]string, 0, len(m.Params))
	values := make([]*emit.Expr, 0, len(m.Params))
	for i, p := range m.Params {
		keys = append(keys, paramFieldName(p.Name, i))
		values = append(values, emit.NewIdent(paramAccessName(p.Name, i)))
	}
	composite := emit.NewCompositeKeyed(emit.Internal(callStruct), keys, values)
	appendCall := emit.NewCall(emit.NewIdent("append"), target, composite)
	stmt := emit.NewAssign(
		[]*emit.Expr{target},
		"=",
		[]*emit.Expr{appendCall},
	)
	return m.Prebody().Append(stmt, emit.Provenance{SetBy: Name})
}

// paramFieldName returns the exported field-name form of a source
// parameter — first-letter-uppercased, or `Arg<N>` for anonymous
// params. The field-name and value-expression sides use the same
// derivation so a missing param name produces matching identifiers
// on both halves of the assignment.
func paramFieldName(name string, index int) string {
	if name == "" {
		return "Arg" + strconv.Itoa(index)
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// paramAccessName returns the in-method identifier referring to a
// source parameter — the source name when non-empty, or the
// `arg<N>` fallback the mock plugin uses for anonymous params.
// Must agree with the mock plugin's `paramName` convention so the
// append-call references a real local.
func paramAccessName(name string, index int) string {
	if name == "" {
		return "arg" + strconv.Itoa(index)
	}
	return name
}
