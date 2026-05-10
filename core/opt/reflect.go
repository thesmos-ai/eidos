// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"go.thesmos.sh/eidos/core/naming"
)

// durationType is the reflect.Type of [time.Duration], used by
// [kindOf] to distinguish it from a plain int64 field.
var durationType = reflect.TypeFor[time.Duration]()

// tagKey is the struct-tag key opt recognises. Other tag keys on the
// same field are ignored.
const tagKey = "eidos"

// Reflect derives a [Schema] from example via reflection on its
// struct fields. Pass a zero value of the options type:
//
//	var schema = opt.Reflect(Options{})
//
// Reflect panics on any tag error — invalid tag syntax, unsupported
// field type, conflicting options. Use [ReflectChecked] for runtime
// derivation where a Go error is preferable to a panic.
//
// The function recognises a comma-separated tag value of the form
// `eidos:"name,required,default=X,one_of=A|B|C,desc=…"`:
//
//   - The first comma-separated segment is the option name. An
//     empty name (e.g. `eidos:",required"`) falls back to the
//     snake_case of the Go field name.
//   - `required` marks the option as mandatory.
//   - `default=VALUE` sets a default value (string-form; parsed per
//     [FieldKind] at decode time).
//   - `one_of=A|B|C` restricts the value to the pipe-separated
//     enumeration. Empty entries (e.g. `one_of=A||C`) are dropped.
//   - `desc=TEXT` attaches a documentation description.
//   - `-` as the entire tag value skips the field.
//   - An absent tag derives the option name from the Go field name
//     and applies no additional constraints.
//
// Unexported fields are always skipped.
func Reflect(example any) Schema {
	s, err := ReflectChecked(example)
	if err != nil {
		// init-time programmer error; documented contract.
		panic(err) //nolint:forbidigo
	}
	return s
}

// ReflectChecked is the error-returning sibling of [Reflect].
func ReflectChecked(example any) (Schema, error) {
	t := reflect.TypeOf(example)
	if t == nil {
		return Schema{}, fmt.Errorf("%w: nil example", ErrInvalidTag)
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return Schema{}, fmt.Errorf("%w: expected struct, got %s", ErrInvalidTag, t.Kind())
	}
	fields := make([]Field, 0, t.NumField())
	for sf := range t.Fields() {
		if !sf.IsExported() {
			continue
		}
		tag, hasTag := sf.Tag.Lookup(tagKey)
		if hasTag && tag == "-" {
			continue
		}
		field, err := parseTag(sf, tag)
		if err != nil {
			return Schema{}, err
		}
		fields = append(fields, field)
	}
	return Schema{Fields: fields}, nil
}

// parseTag turns the `eidos:` tag content for one struct field into
// a populated [Field]. It validates option names and unknown tag
// segments by wrapping them with [ErrInvalidTag].
func parseTag(sf reflect.StructField, tag string) (Field, error) {
	field := Field{GoFieldName: sf.Name}

	kind, err := kindOf(sf.Type)
	if err != nil {
		return Field{}, fmt.Errorf("%w (field %s of type %s)", err, sf.Name, sf.Type)
	}
	field.Kind = kind

	if tag == "" {
		field.Name = naming.Snake(sf.Name)
		return field, nil
	}

	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		field.Name = naming.Snake(sf.Name)
	} else {
		field.Name = parts[0]
	}

	for _, opt := range parts[1:] {
		if err := applyTagOption(&field, opt, sf.Name); err != nil {
			return Field{}, err
		}
	}
	if err := validateField(field, sf.Name); err != nil {
		return Field{}, err
	}
	return field, nil
}

// validateField checks the Reflect-time invariants for one field:
// OneOf is only meaningful for KindString; a declared Default must
// parse cleanly for the field's [FieldKind]; a declared Default
// combined with a OneOf list must be a member of that list.
//
// All failures wrap [ErrInvalidTag] so plugin-author misconfiguration
// fails at init / build time, not at the first config decode.
func validateField(f Field, goFieldName string) error {
	if len(f.OneOf) > 0 && f.Kind != KindString {
		return fmt.Errorf("%w: one_of is only valid for string fields (got %s on %s)",
			ErrInvalidTag, f.Kind, goFieldName)
	}
	if !f.HasDefault {
		return nil
	}
	if len(f.OneOf) > 0 && !slices.Contains(f.OneOf, f.DefaultStr) {
		return fmt.Errorf("%w: default %q for %s is not in one_of %v",
			ErrInvalidTag, f.DefaultStr, goFieldName, f.OneOf)
	}
	if _, err := parseValue(f.DefaultStr, f.Kind, nil); err != nil {
		return fmt.Errorf("%w: default for %s: %w", ErrInvalidTag, goFieldName, err)
	}
	return nil
}

// applyTagOption applies a single tag option (e.g. "required" or
// "default=Foo") to field. Unknown options return [ErrInvalidTag].
func applyTagOption(field *Field, opt, fieldName string) error {
	switch {
	case opt == "required":
		field.Required = true
	case strings.HasPrefix(opt, "default="):
		field.HasDefault = true
		field.DefaultStr = strings.TrimPrefix(opt, "default=")
	case strings.HasPrefix(opt, "one_of="):
		list := strings.TrimPrefix(opt, "one_of=")
		for v := range strings.SplitSeq(list, "|") {
			if v != "" {
				field.OneOf = append(field.OneOf, v)
			}
		}
	case strings.HasPrefix(opt, "desc="):
		field.Description = strings.TrimPrefix(opt, "desc=")
	default:
		return fmt.Errorf("%w: unknown tag option %q on field %s", ErrInvalidTag, opt, fieldName)
	}
	return nil
}

// kindOf maps a reflect.Type to the matching [FieldKind] for
// supported types. [time.Duration] is recognised before the generic
// int64 fall-through so it parses via [time.ParseDuration] instead of
// [strconv.Atoi]. Unsupported types yield [ErrUnsupportedFieldType].
func kindOf(t reflect.Type) (FieldKind, error) {
	if t == durationType {
		return KindDuration, nil
	}
	switch t.Kind() {
	case reflect.String:
		return KindString, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return KindInt, nil
	case reflect.Bool:
		return KindBool, nil
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String {
			return KindStringList, nil
		}
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedFieldType, t)
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedFieldType, t)
	}
}
