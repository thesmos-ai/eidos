// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/kind"
)

// Tag is one entry in a [Field]'s struct-tag block (Go's
// backtick-wrapped tag literal: `json:"name" db:"name"`). Each Tag
// is a key + value pair; the backend renders them as
// `Key:"EscapedValue"` and joins multiple entries with single
// spaces.
//
// Cross-cutting generators (json-tag, db-tag, validation-tag
// plugins, …) append [*Tag] entries to a Field's tags slot via
// [Field.Tags]. The host generator may also declare a base tag via
// [Field.Tag] (a raw string written directly into the backticks);
// the backend renders the base tag first, then the slot's entries
// in append order.
//
// Tag values are escaped per Go-string rules — the backend wraps
// each Value in double quotes via [strconv.Quote] equivalence so
// embedded quotes, backslashes, and non-printable characters
// survive the round-trip.
type Tag struct {
	BaseEmit

	// Key is the tag-namespace identifier ("json", "db", "yaml",
	// "validate", …). Conventionally a short lowercase ident; the
	// backend does not validate the value.
	Key string `json:"key"`

	// Value is the unquoted tag value. The backend re-quotes via
	// Go-string-escape rules at render time.
	Value string `json:"value"`
}

// Kind returns [KindTag].
func (*Tag) Kind() kind.Kind { return KindTag }
