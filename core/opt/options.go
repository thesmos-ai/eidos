// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Options is a typed view onto raw configuration values for a single
// plugin. The pipeline builds an Options by pairing a [Schema]
// (derived from the plugin's struct tags via [Reflect]) with the
// raw `map[string]string` values supplied by whatever config-format
// loader is in use.
//
// Options is intentionally minimal: [Options.Decode] is the one
// operation plugin code performs against it. Construct via [New];
// the zero value is unusable.
type Options struct {
	schema Schema
	values map[string]string
}

// New returns an Options binding schema to values. Neither argument is
// validated at construction time — validation is deferred to
// [Options.Decode] so every problem surfaces in one place with the
// destination struct's context.
//
// The values map is held by reference; callers should not mutate it
// after constructing the Options.
func New(schema Schema, values map[string]string) Options {
	return Options{schema: schema, values: values}
}

// Schema returns the schema this Options is bound to.
func (o Options) Schema() Schema { return o.schema }

// Has reports whether the input map carries the named option.
// Defaults are not consulted; use [Options.Decode] for full
// "post-default" semantics.
func (o Options) Has(name string) bool {
	_, ok := o.values[name]
	return ok
}

// Decode validates the bound values against the schema, applies
// declared defaults to absent optional fields, and populates dst (a
// non-nil pointer to a struct of the type the schema was derived
// from). Returns nil on success.
//
// Error families:
//
//   - [ErrInvalidDecodeTarget]: dst is not a pointer to a struct.
//   - [ErrMissingRequired]: a required option was not supplied.
//   - [ErrInvalidValue]: a value failed per-kind parsing or fell
//     outside its [Field.OneOf] enumeration.
//   - [ErrUnknownField]: the input map contains a key the schema
//     does not declare.
//
// Decode is strict about unknown keys: silent drops would mask
// config-file typos, which the pipeline surfaces as positioned
// diagnostics rather than ignoring.
func (o Options) Decode(dst any) error {
	target, err := derefStructTarget(dst)
	if err != nil {
		return err
	}
	if err := o.rejectUnknownFields(); err != nil {
		return err
	}
	for _, f := range o.schema.Fields {
		raw, present := o.values[f.Name]
		if !present {
			if f.Required {
				return fmt.Errorf("%w: %s", ErrMissingRequired, f.Name)
			}
			if !f.HasDefault {
				continue
			}
			raw = f.DefaultStr
		}
		if err := assignField(target, f, raw); err != nil {
			return err
		}
	}
	return nil
}

// rejectUnknownFields returns [ErrUnknownField] for the first input
// key the schema does not declare. Returns nil when every input key
// is recognised.
func (o Options) rejectUnknownFields() error {
	for k := range o.values {
		if !o.schema.HasField(k) {
			return fmt.Errorf("%w: %q", ErrUnknownField, k)
		}
	}
	return nil
}

// derefStructTarget validates that dst is a non-nil pointer to a
// struct and returns the addressable struct Value.
func derefStructTarget(dst any) (reflect.Value, error) {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return reflect.Value{}, fmt.Errorf("%w: nil or non-pointer", ErrInvalidDecodeTarget)
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf(
			"%w: pointer must address a struct (got %s)",
			ErrInvalidDecodeTarget, elem.Kind(),
		)
	}
	return elem, nil
}

// assignField parses raw per the field's kind and OneOf and sets the
// destination struct's matching Go field. Returns [ErrInvalidValue]
// for parse / enumeration failures.
func assignField(target reflect.Value, f Field, raw string) error {
	val, err := parseValue(raw, f.Kind, f.OneOf)
	if err != nil {
		return fmt.Errorf("field %s: %w", f.Name, err)
	}
	dst := target.FieldByName(f.GoFieldName)
	switch f.Kind {
	case KindString:
		dst.SetString(val.(string))
	case KindInt:
		dst.SetInt(int64(val.(int)))
	case KindBool:
		dst.SetBool(val.(bool))
	case KindStringList:
		dst.Set(reflect.ValueOf(val.([]string)))
	case KindDuration:
		dst.SetInt(int64(val.(time.Duration)))
	}
	return nil
}

// parseValue converts the raw string form of an option value into
// the typed Go value matching kind. The oneOf check, when non-empty,
// runs before the type parse so OneOf failures take precedence over
// parse failures.
//
// Returned types per kind:
//
//   - KindString       → string
//   - KindInt          → int
//   - KindBool         → bool
//   - KindStringList   → []string (empty input yields []string{})
//   - KindDuration     → time.Duration
func parseValue(raw string, kind FieldKind, oneOf []string) (any, error) {
	if len(oneOf) > 0 && !slices.Contains(oneOf, raw) {
		return nil, fmt.Errorf("%w: %q not in %v", ErrInvalidValue, raw, oneOf)
	}
	switch kind {
	case KindString:
		return raw, nil
	case KindInt:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: int %q: %w", ErrInvalidValue, raw, err)
		}
		return n, nil
	case KindBool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: bool %q: %w", ErrInvalidValue, raw, err)
		}
		return b, nil
	case KindStringList:
		if raw == "" {
			return []string{}, nil
		}
		return strings.Split(raw, ","), nil
	case KindDuration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: duration %q: %w", ErrInvalidValue, raw, err)
		}
		return d, nil
	default:
		return nil, fmt.Errorf("%w: unknown kind %v", ErrInvalidValue, kind)
	}
}
