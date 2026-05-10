// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt

// Field describes one option's contract — its public name, the Go
// struct field it decodes into, its [FieldKind], and any validation
// constraints declared via tags.
//
// Construct fields indirectly via [Reflect] / [ReflectChecked]; a
// hand-built [Schema] may use struct literals when no tagged Go
// struct backs the schema (rare; primarily a test affordance).
type Field struct {
	// Name is the public option name — what the config file or
	// programmatic Options uses as the key. Typically the
	// snake_case form of the Go field name unless overridden by the
	// `eidos:` tag.
	Name string
	// GoFieldName is the destination struct field name. Decode uses
	// reflection to set this field by name.
	GoFieldName string
	// Kind describes the Go type of the destination field.
	Kind FieldKind
	// Required marks the option as mandatory. Decode returns
	// [ErrMissingRequired] when a required field is absent.
	Required bool
	// HasDefault reports whether DefaultStr has been set. Distinct
	// from `DefaultStr == ""` so an explicit empty default is
	// distinguishable from "no default".
	HasDefault bool
	// DefaultStr is the string form of the default. Decode parses it
	// per [Kind] when the input does not supply the field.
	DefaultStr string
	// OneOf, when non-empty, restricts the permitted string values.
	// Only meaningful for KindString; KindInt and the others ignore
	// it.
	OneOf []string
	// Description is for documentation generation; validation does
	// not consult it.
	Description string
}
