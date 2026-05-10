// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt

// Schema is the contract for a plugin's options — the ordered set of
// [Field] declarations the options struct exposes. Schemas are value
// types and safe to share across goroutines.
//
// Construct via [Reflect] (panics on tag errors) or
// [ReflectChecked] (returns errors). Direct struct-literal use is
// supported for tests and hand-coded schemas, but most plugins
// reflect off a tagged Go struct.
type Schema struct {
	Fields []Field
}

// Lookup returns the [Field] with the given option name and reports
// whether it was found. Name comparison is case-sensitive.
func (s Schema) Lookup(name string) (Field, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

// HasField reports whether the schema declares an option with the
// given name. Equivalent to discarding the field from [Schema.Lookup].
func (s Schema) HasField(name string) bool {
	_, ok := s.Lookup(name)
	return ok
}

// Names returns every declared option name in declaration order.
func (s Schema) Names() []string {
	out := make([]string, 0, len(s.Fields))
	for _, f := range s.Fields {
		out = append(out, f.Name)
	}
	return out
}
