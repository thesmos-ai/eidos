// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo

// Go-side scalar identifiers — kept as named constants because
// each appears multiple times in the scalar translation table.
const (
	goInt32  = "int32"
	goInt64  = "int64"
	goUint32 = "uint32"
	goUint64 = "uint64"
)

// scalarTable maps proto-native scalar names (as the frontend
// records them on [node.TypeRef.Name]) to their Go-side rendered
// form. The mapping mirrors the protobuf-Go canonical generator's
// scalar choices: signed integer widths preserve their width
// (`int32`/`int64`), unsigned widths use the matching Go-uint
// type, `float`/`double` map to `float32`/`float64`, `bytes`
// maps to `[]byte`, and `bool`/`string` pass through verbatim.
//
//nolint:gochecknoglobals // immutable scalar lookup table.
var scalarTable = map[string]string{
	"bool":     "bool",
	"string":   "string",
	"bytes":    "[]byte",
	"int32":    goInt32,
	"sint32":   goInt32,
	"sfixed32": goInt32,
	"int64":    goInt64,
	"sint64":   goInt64,
	"sfixed64": goInt64,
	"uint32":   goUint32,
	"fixed32":  goUint32,
	"uint64":   goUint64,
	"fixed64":  goUint64,
	"float":    "float32",
	"double":   "float64",
}

// scalarGoType returns the Go-side form for a proto scalar name,
// or empty when name is not a known scalar (the caller then
// dispatches on well-known + named-message cases).
func scalarGoType(name string) string {
	return scalarTable[name]
}

// wellKnownTable maps well-known proto type bare names (the
// values [protobuf.MetaWellKnown] stamps) to their Go-side
// rendered form. The mapping follows the canonical
// `google.golang.org/protobuf/types/known/<x>pb` package layout:
// every well-known surfaces as a pointer reference because the
// generated Go types are pointer-receivers.
//
//nolint:gochecknoglobals // immutable well-known lookup table.
var wellKnownTable = map[string]string{
	"Timestamp":   "*timestamppb.Timestamp",
	"Duration":    "*durationpb.Duration",
	"Empty":       "*emptypb.Empty",
	"Any":         "*anypb.Any",
	"FieldMask":   "*fieldmaskpb.FieldMask",
	"Struct":      "*structpb.Struct",
	"Value":       "*structpb.Value",
	"ListValue":   "*structpb.ListValue",
	"NullValue":   "structpb.NullValue",
	"BoolValue":   "*wrapperspb.BoolValue",
	"StringValue": "*wrapperspb.StringValue",
	"Int32Value":  "*wrapperspb.Int32Value",
	"Int64Value":  "*wrapperspb.Int64Value",
	"UInt32Value": "*wrapperspb.UInt32Value",
	"UInt64Value": "*wrapperspb.UInt64Value",
	"FloatValue":  "*wrapperspb.FloatValue",
	"DoubleValue": "*wrapperspb.DoubleValue",
	"BytesValue":  "*wrapperspb.BytesValue",
}

// wellKnownGoType returns the Go-side form for a well-known
// type's bare name, or empty when the name isn't in the
// canonical well-known set.
func wellKnownGoType(name string) string {
	return wellKnownTable[name]
}

// wellKnownPackagePrefix is the common path prefix every
// well-known type's generated Go package lives under. Each entry
// in [wellKnownImports] composes its final import path by
// appending a per-family suffix to this prefix.
const wellKnownPackagePrefix = "google.golang.org/protobuf/types/known/"

// wellKnownImports maps each well-known bare name to the Go
// import path the generated reference lives at. The render-site
// registers this import on the host file's ImportSet when the
// bridge's go.type override fires for a well-known. Paths share
// [wellKnownPackagePrefix] so the table reads as a per-name
// family suffix.
//
//nolint:gochecknoglobals // immutable lookup table.
var wellKnownImports = map[string]string{
	"Timestamp":   wellKnownPackagePrefix + "timestamppb",
	"Duration":    wellKnownPackagePrefix + "durationpb",
	"Empty":       wellKnownPackagePrefix + "emptypb",
	"Any":         wellKnownPackagePrefix + "anypb",
	"FieldMask":   wellKnownPackagePrefix + "fieldmaskpb",
	"Struct":      wellKnownPackagePrefix + "structpb",
	"Value":       wellKnownPackagePrefix + "structpb",
	"ListValue":   wellKnownPackagePrefix + "structpb",
	"NullValue":   wellKnownPackagePrefix + "structpb",
	"BoolValue":   wellKnownPackagePrefix + "wrapperspb",
	"StringValue": wellKnownPackagePrefix + "wrapperspb",
	"Int32Value":  wellKnownPackagePrefix + "wrapperspb",
	"Int64Value":  wellKnownPackagePrefix + "wrapperspb",
	"UInt32Value": wellKnownPackagePrefix + "wrapperspb",
	"UInt64Value": wellKnownPackagePrefix + "wrapperspb",
	"FloatValue":  wellKnownPackagePrefix + "wrapperspb",
	"DoubleValue": wellKnownPackagePrefix + "wrapperspb",
	"BytesValue":  wellKnownPackagePrefix + "wrapperspb",
}

// wellKnownImport returns the Go import path for a well-known
// type's bare name, or empty when the name isn't in the canonical
// well-known set.
func wellKnownImport(name string) string {
	return wellKnownImports[name]
}
