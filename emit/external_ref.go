// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

import (
	"go.thesmos.sh/eidos/core/kind"
)

// ExternalRef references a third-party or stdlib type by its package
// path and unqualified name. The backend's `imp` template func
// translates ExternalRefs into the correct local alias at render
// time and registers the path with the file's ImportSet.
//
//	ctxRef := emit.External("context", "Context")
//	dbRef  := emit.External("database/sql", "DB")
//
// TypeArgs holds generic instantiation arguments where applicable.
type ExternalRef struct {
	BaseEmit
	Package  string `json:"package"`
	Name     string `json:"name"`
	TypeArgs []Ref  `json:"-"`
}

// Kind returns [KindExternalRef].
func (*ExternalRef) Kind() kind.Kind { return KindExternalRef }

// isRef marks ExternalRef as a [Ref] implementation.
func (*ExternalRef) isRef() {}

// External constructs an ExternalRef for the supplied package path
// and type name. Pass optional TypeArgs for generic instantiation.
func External(pkg, name string, typeArgs ...Ref) *ExternalRef {
	return &ExternalRef{Package: pkg, Name: name, TypeArgs: typeArgs}
}
