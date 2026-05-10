// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt

import "strconv"

// FieldKind enumerates the Go types a [Field] may carry. The set is
// intentionally narrow: strings, ints, bools, and string lists cover
// every plugin-option use case seen in practice without smuggling
// language-specific types into the schema.
type FieldKind int

// The kinds in source-declaration order. Add new kinds at the end so
// existing ordinals stay stable for any consumer that persists them.
const (
	// KindString matches Go's string type.
	KindString FieldKind = iota
	// KindInt matches int, int8, int16, int32, int64 — all decoded
	// via strconv.Atoi into the destination field's natural width.
	KindInt
	// KindBool matches Go's bool type. Accepted string forms are
	// "true"/"false"/"1"/"0" via strconv.ParseBool.
	KindBool
	// KindStringList matches []string. Values are split on ","; an
	// empty input decodes to an empty slice (not nil) so "present
	// but empty" round-trips cleanly.
	KindStringList
	// KindDuration matches [time.Duration]. Values are parsed via
	// [time.ParseDuration] ("30s", "5m", "1h30m").
	KindDuration
)

// String returns the lower-case textual form of k for diagnostics.
// Unknown values stringify as "kind(N)" rather than panicking.
func (k FieldKind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindBool:
		return "bool"
	case KindStringList:
		return "string_list"
	case KindDuration:
		return "duration"
	default:
		return "kind(" + strconv.Itoa(int(k)) + ")"
	}
}
