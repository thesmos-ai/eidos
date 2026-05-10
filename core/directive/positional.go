// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive

// PositionalArg describes one positional argument slot in a [Schema].
//
// Name is for diagnostic clarity ("Got %d positional args; expected
// 'target' at position 0"). Required = true blocks the directive
// when the slot is missing; Required = false plus Default populates
// the slot during [Validate]. OneOf, when non-empty, restricts the
// permitted values; an empty OneOf accepts any value. Description
// feeds the (not-yet-shipped) directive documentation generator.
type PositionalArg struct {
	Name        string
	Required    bool
	OneOf       []string
	Default     string
	Description string
}

// PositionalOption is a functional option applied by [SchemaBuilder.Positional].
type PositionalOption func(*PositionalArg)

// Required marks the positional slot as mandatory.
func Required() PositionalOption {
	return func(p *PositionalArg) { p.Required = true }
}

// OneOf restricts the value of the positional slot to one of the
// given strings. Values are compared verbatim; case-sensitivity is
// the caller's responsibility.
func OneOf(values ...string) PositionalOption {
	return func(p *PositionalArg) { p.OneOf = append(p.OneOf, values...) }
}

// Default sets the value used when a non-required positional slot
// is missing. Default on a Required slot is a no-op — the slot is
// rejected before any default would apply.
func Default(v string) PositionalOption {
	return func(p *PositionalArg) { p.Default = v }
}

// Describe attaches a human-readable description for documentation
// generation. Validation does not consult Description; it is purely
// informational.
func Describe(s string) PositionalOption {
	return func(p *PositionalArg) { p.Description = s }
}
