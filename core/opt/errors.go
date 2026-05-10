// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package opt

import "errors"

// ErrInvalidTag is returned by [ReflectChecked] when an `eidos:` tag
// is malformed or references an unknown option key. Programmer-error
// territory: surface during build / test, not at runtime.
var ErrInvalidTag = errors.New("opt: invalid eidos tag")

// ErrUnsupportedFieldType is returned by [ReflectChecked] when an
// options struct field has a Go type the package does not handle.
var ErrUnsupportedFieldType = errors.New("opt: unsupported field type")

// ErrInvalidDecodeTarget is returned by [Options.Decode] when the
// destination passed in is not a pointer to a struct.
var ErrInvalidDecodeTarget = errors.New("opt: Decode requires a pointer to struct")

// ErrMissingRequired is returned by [Options.Decode] when a Field
// with Required=true was not supplied in the input map.
var ErrMissingRequired = errors.New("opt: missing required option")

// ErrInvalidValue is returned by [Options.Decode] for values that
// fail per-kind parsing (e.g. non-numeric input for KindInt) or fall
// outside the field's OneOf enumeration.
var ErrInvalidValue = errors.New("opt: invalid option value")

// ErrUnknownField is returned by [Options.Decode] when the input
// map contains a key the schema does not declare. Strict-by-default
// behaviour catches typos at the config-file source rather than
// silently dropping them.
var ErrUnknownField = errors.New("opt: unknown option")
