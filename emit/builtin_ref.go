// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/kind"
)

// BuiltinRef references a language built-in (int, string, bool,
// any, …) — a type the target language understands without any
// import. Backends render BuiltinRefs by name only.
//
// The set of valid Name values is language-specific. The emit layer
// does not validate; the backend is responsible for emitting a
// recognisable builtin name (gofmt-clean for Go, rustfmt-clean for
// Rust, …).
type BuiltinRef struct {
	BaseEmit
	Name string `json:"name"`
}

// Kind returns [KindBuiltinRef].
func (*BuiltinRef) Kind() kind.Kind { return KindBuiltinRef }

// isRef marks BuiltinRef as a [Ref] implementation.
func (*BuiltinRef) isRef() {}

// Builtin constructs a BuiltinRef with the given name. Common
// callers: emit.Builtin("int"), emit.Builtin("string"),
// emit.Builtin("error"), emit.Builtin("any").
func Builtin(name string) *BuiltinRef {
	return &BuiltinRef{Name: name}
}
