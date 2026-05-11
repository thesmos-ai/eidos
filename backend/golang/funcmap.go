// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"errors"
	"fmt"

	"go.thesmos.sh/eidos/emit"
)

// ErrUnsupportedRef is returned by [renderType] when called with an
// [emit.Ref] kind the current funcmap can't render. The wrapped
// message names the concrete Go type so the offending generator is
// identifiable from the diagnostic alone. Type-ref support widens
// as additional [emit.Ref] implementations (TypeRef, ExternalRef,
// the various [emit.CompositeRef] shapes) are wired into the
// backend.
var ErrUnsupportedRef = errors.New("backend/golang: renderType: unsupported Ref")

// renderType produces the Go source spelling for r. Supported kinds
// today: [emit.BuiltinRef] (rendered as its name). Other ref kinds
// return [ErrUnsupportedRef] wrapped with the concrete Go type so
// diagnostics attribute the gap precisely.
//
// `renderType` is one of the reserved core funcmap entries —
// plugin overrides for this name are rejected at Build time.
func renderType(r emit.Ref) (string, error) {
	switch typed := r.(type) {
	case *emit.BuiltinRef:
		return typed.Name, nil
	default:
		return "", fmt.Errorf("%w: %T", ErrUnsupportedRef, r)
	}
}
