// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
)

// stampFileOptions walks fd's file-level options and stamps each
// set option on pkg's meta bag under the documented
// `proto.option.<full-name>` channel. Standard options (defined
// on google.protobuf.FileOptions) stamp under their short name —
// `proto.option.go_package`, `proto.option.deprecated`. Custom
// extension options stamp under the extension's proto FQN —
// `proto.option.acme.cfg.feature_flag`. Value types follow the
// documented value-type mapping.
//
// File-level options accumulate across multiple .proto files
// sharing one proto package: later files' options merge into the
// already-stamped package bag. The merge is order-deterministic
// because [convertFiles] sorts descriptors alphabetically by file
// path before invoking this helper. Same-key cross-file
// collisions on string- or bool-typed options surface as a Warn
// so users see the silent last-writer-wins behaviour rather than
// inheriting one file's value at random.
//
// Non-file hosts (message, field, oneof, enum, enum-variant,
// service, method) route through [stampHostOptions] instead —
// each descriptor owns its options at one site, so the
// cross-source overwrite-warn is file-specific. Both paths
// converge on [stampOptionOnBag] for the per-option stamping
// itself, so the value-type table and meta-key namespace are
// shared.
func stampFileOptions(ps *diag.PluginSink, pkg *node.Package, fd protoreflect.FileDescriptor) {
	opts := fd.Options()
	if opts == nil {
		return
	}
	pos := position.Pos{File: fd.Path()}
	opts.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		stampFileOption(ps, pkg, fd, field, value, pos)
		return true
	})
}

// stampFileOption stamps one file-level option pair with the
// cross-file overwrite-warn guard layered on top of the generic
// path. Detects collisions for the common scalar forms (string,
// bool); other forms route through the same generic path
// without an overwrite check (the file-level collision risk is
// effectively scalar-shaped — every well-known FileOptions field
// is either a string, bool, or enum).
func stampFileOption(
	ps *diag.PluginSink, pkg *node.Package, fd protoreflect.FileDescriptor,
	field protoreflect.FieldDescriptor, value protoreflect.Value, pos position.Pos,
) {
	name := optionKeyName(field)
	switch field.Kind() { //nolint:exhaustive // overwrite warn covers scalar-shaped FileOptions kinds only
	case protoreflect.StringKind:
		warnIfFileOptionOverwrites(
			ps, pkg, name, value.String(), fd.Path(), pos, "%q",
		)
	case protoreflect.BoolKind:
		warnIfFileOptionOverwrites(
			ps, pkg, name, value.Bool(), fd.Path(), pos, "%t",
		)
	}
	stampOptionOnBag(ps, pkg.Meta(), field, value, pos)
}

// warnIfFileOptionOverwrites detects a same-key cross-file
// overwrite on a scalar option and surfaces a positioned Warn
// describing the prior/new pair. The generic format string lets
// the same helper handle string-quoted and bool-natural rendering
// without per-type duplication.
func warnIfFileOptionOverwrites[T comparable](
	ps *diag.PluginSink, pkg *node.Package,
	name string, value T, file string, pos position.Pos, valueFormat string,
) {
	key, ok := lookupScalarOptionKey[T](name)
	if !ok {
		return
	}
	prior, ok := key.Get(pkg.Meta())
	if !ok || prior == value {
		return
	}
	ps.Warnf(
		pos,
		fmt.Sprintf(
			"protobuf: file option %%s = %s on %%s overwrites prior value %s from sibling file",
			valueFormat, valueFormat,
		),
		name, value, file, prior,
	)
}

// lookupScalarOptionKey returns the typed [meta.Key] for the
// option name under the [MetaOptionPrefix] namespace, dispatching
// the parser by T. Returns false for value types the file-level
// overwrite path doesn't track.
func lookupScalarOptionKey[T comparable](name string) (meta.Key[T], bool) {
	full := MetaOptionPrefix + name
	switch any(*new(T)).(type) {
	case string:
		k := meta.EnsureKey(full, meta.StringParser)
		return any(k).(meta.Key[T]), true
	case bool:
		k := meta.EnsureKey(full, meta.BoolParser)
		return any(k).(meta.Key[T]), true
	}
	return meta.Key[T]{}, false
}

// stampHostOptions walks optsMsg and stamps each set option on
// bag under [MetaOptionPrefix] + the option's name. Returns
// without writing when optsMsg is nil or its reflection surface
// is the empty-message form. Used for every non-file host kind
// (message, field, oneof, enum, enum-variant, service, method) —
// each carries its own descriptorpb.*Options message via
// protoreflect.Options. The file-level counterpart
// [stampFileOptions] uses the same per-option stamping path via
// [stampOptionOnBag] but layers a cross-file overwrite-warn on
// top.
func stampHostOptions(
	ps *diag.PluginSink, bag *meta.Bag,
	optsMsg proto.Message, pos position.Pos,
) {
	if optsMsg == nil {
		return
	}
	refl := optsMsg.ProtoReflect()
	if !refl.IsValid() {
		return
	}
	refl.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		stampOptionOnBag(ps, bag, field, value, pos)
		return true
	})
}

// stampOptionOnBag stamps one option (field, value) pair on bag.
// The value's natural Go form is determined by [optionValueToGo]
// and stored under the per-type [meta.Key]. Unsupported value
// forms surface as a positioned Warn rather than a silent drop.
func stampOptionOnBag(
	ps *diag.PluginSink, bag *meta.Bag,
	field protoreflect.FieldDescriptor, value protoreflect.Value, pos position.Pos,
) {
	name := optionKeyName(field)
	typed, ok := optionValueToGo(field, value)
	if !ok {
		ps.Warnf(
			pos,
			"protobuf: option %s carries unsupported value kind %s; skipped",
			name, field.Kind(),
		)
		return
	}
	stampTypedOption(bag, name, typed, pos)
}

// optionValueToGo returns the natural Go form of value per the
// documented value-type table. Map fields produce map[string]any
// keyed by stringified K; repeated fields produce []any in
// source-declaration order; scalars produce their natural Go
// type (string, bool, int64, uint64, float64, []byte); enums
// produce the variant name string; messages produce a recursive
// map[string]any keyed by proto field name.
//
// Returns (nil, false) for value forms outside the table — the
// caller surfaces the case as a Warn.
func optionValueToGo(field protoreflect.FieldDescriptor, value protoreflect.Value) (any, bool) {
	switch {
	case field.IsMap():
		return mapValueToGo(field, value.Map()), true
	case field.IsList():
		return listValueToGo(field, value.List()), true
	default:
		return scalarValueToGo(field, value)
	}
}

// scalarValueToGo returns the Go form of a singular (non-list,
// non-map) field. Documented value-type widening collapses every
// signed-int width to int64, every unsigned-int width to uint64,
// and float/double to float64. Enum values resolve to the
// variant's Name string; message values recurse via
// [messageValueToGo].
func scalarValueToGo(field protoreflect.FieldDescriptor, value protoreflect.Value) (any, bool) {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return value.Bool(), true
	case protoreflect.StringKind:
		return value.String(), true
	case protoreflect.BytesKind:
		return value.Bytes(), true
	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return value.Int(), true
	case protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return value.Uint(), true
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return value.Float(), true
	case protoreflect.EnumKind:
		name := enumVariantName(field, value.Enum())
		if name == "" {
			return nil, false
		}
		return name, true
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return messageValueToGo(value.Message()), true
	default:
		return nil, false
	}
}

// listValueToGo returns the []any form of a repeated option
// value. Each element runs through [scalarValueToGo]; failed
// conversions surface as nil placeholders so the slice still
// has one entry per source element.
func listValueToGo(field protoreflect.FieldDescriptor, list protoreflect.List) []any {
	count := list.Len()
	out := make([]any, 0, count)
	for i := range count {
		v, _ := scalarValueToGo(field, list.Get(i))
		out = append(out, v)
	}
	return out
}

// mapValueToGo returns the map[string]any form of a map<K, V>
// option value. Keys stringify through [protoreflect.MapKey.String];
// values run through [scalarValueToGo] against the entry
// message's value field descriptor.
func mapValueToGo(field protoreflect.FieldDescriptor, m protoreflect.Map) map[string]any {
	out := map[string]any{}
	valField := field.MapValue()
	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		val, _ := scalarValueToGo(valField, v)
		out[k.String()] = val
		return true
	})
	return out
}

// messageValueToGo recursively converts a message-typed option
// value into a map[string]any keyed by proto field name. Each
// populated field routes through [optionValueToGo] so nested
// messages, repeated fields, and maps round-trip into nested
// Go containers.
func messageValueToGo(msg protoreflect.Message) map[string]any {
	out := map[string]any{}
	msg.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if v, ok := optionValueToGo(field, value); ok {
			out[string(field.Name())] = v
		}
		return true
	})
	return out
}

// stampTypedOption picks the right [meta.Key] based on the
// runtime type of value and stamps it on bag. Each option name
// resolves to one Go type by virtue of the proto schema; once a
// key registers, subsequent stamps with the same name must match
// the registered T (the framework's [meta.EnsureKey] enforces
// this).
func stampTypedOption(bag *meta.Bag, name string, value any, pos position.Pos) {
	full := MetaOptionPrefix + name
	switch v := value.(type) {
	case string:
		meta.EnsureKey(full, meta.StringParser).SetAt(bag, v, meta.AuthorityPlugin, FrontendName, pos)
	case bool:
		meta.EnsureKey(full, meta.BoolParser).SetAt(bag, v, meta.AuthorityPlugin, FrontendName, pos)
	case int64:
		meta.EnsureKey(full, Int64Parser).SetAt(bag, v, meta.AuthorityPlugin, FrontendName, pos)
	case uint64:
		meta.EnsureKey(full, Uint64Parser).SetAt(bag, v, meta.AuthorityPlugin, FrontendName, pos)
	case float64:
		meta.EnsureKey(full, Float64Parser).SetAt(bag, v, meta.AuthorityPlugin, FrontendName, pos)
	case []byte:
		meta.EnsureKey(full, BytesParser).SetAt(bag, v, meta.AuthorityPlugin, FrontendName, pos)
	case map[string]any:
		meta.EnsureKey(full, MapAnyParser).SetAt(bag, v, meta.AuthorityPlugin, FrontendName, pos)
	case []any:
		meta.EnsureKey(full, SliceAnyParser).SetAt(bag, v, meta.AuthorityPlugin, FrontendName, pos)
	}
}

// Int64Parser decodes a JSON-encoded int64. The parser is
// invoked on the cache-roundtrip path where the bag was
// JSON-marshalled and the original int64 lazy-decodes here.
func Int64Parser(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	var out int64
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return 0, fmt.Errorf("protobuf: decode int64: %w", err)
	}
	return out, nil
}

// Uint64Parser decodes a JSON-encoded uint64.
func Uint64Parser(raw string) (uint64, error) {
	if raw == "" {
		return 0, nil
	}
	var out uint64
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return 0, fmt.Errorf("protobuf: decode uint64: %w", err)
	}
	return out, nil
}

// Float64Parser decodes a JSON-encoded float64.
func Float64Parser(raw string) (float64, error) {
	if raw == "" {
		return 0, nil
	}
	var out float64
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return 0, fmt.Errorf("protobuf: decode float64: %w", err)
	}
	return out, nil
}

// BytesParser decodes a JSON-encoded []byte (base64-encoded
// per Go's stdlib).
func BytesParser(raw string) ([]byte, error) {
	if raw == "" {
		return nil, nil
	}
	var out []byte
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("protobuf: decode []byte: %w", err)
	}
	return out, nil
}

// MapAnyParser decodes a JSON-encoded map[string]any. Used for
// message-typed and map-typed option values' cache roundtrip.
// Empty input decodes to an empty (non-nil) map so callers can
// distinguish "absent" from "present with no entries".
func MapAnyParser(raw string) (map[string]any, error) {
	out := map[string]any{}
	if raw == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("protobuf: decode map[string]any: %w", err)
	}
	return out, nil
}

// SliceAnyParser decodes a JSON-encoded []any. Used for repeated
// option values' cache roundtrip. Empty input decodes to an
// empty (non-nil) slice so callers can distinguish "absent" from
// "present with no entries".
func SliceAnyParser(raw string) ([]any, error) {
	out := []any{}
	if raw == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("protobuf: decode []any: %w", err)
	}
	return out, nil
}

// enumVariantName resolves an enum-typed option value to the
// variant's source-form name. Returns the empty string when the
// numeric value doesn't match any declared variant — protocompile
// normally rejects such inputs at parse time; the empty return is
// the defensive fallback callers surface as a Warn.
func enumVariantName(field protoreflect.FieldDescriptor, number protoreflect.EnumNumber) string {
	v := field.Enum().Values().ByNumber(number)
	if v == nil {
		return ""
	}
	return string(v.Name())
}
