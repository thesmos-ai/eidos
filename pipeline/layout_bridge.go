// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
)

// bridgeGoNameKey is the bridge-stamped `go.name` meta key the
// layout consults when resolving a rendered file's package
// clause. Shared cross-package via [meta.EnsureKey] so the
// layout doesn't import any bridge plugin's exported constants.
//
//nolint:gochecknoglobals // cross-package registry-singleton key
var bridgeGoNameKey = meta.EnsureKey("go.name", meta.StringParser)

// bridgeGoImportKey is the bridge-stamped `go.import` meta key
// the layout consults when resolving a rendered file's import
// path. Same cross-package singleton convention as
// [bridgeGoNameKey].
//
//nolint:gochecknoglobals // cross-package registry-singleton key
var bridgeGoImportKey = meta.EnsureKey("go.import", meta.StringParser)

// bridgePackageNameOr returns the bridge-stamped `go.name` from
// origin (when origin is a [node.Package] carrying it) or
// fallback when no bridge stamp is present. The lookup lets a
// cross-language bridge annotator (the protogo bridge for
// proto→Go, future protorust / prototypescript variants)
// override the rendered package clause without the layout
// learning anything language-specific.
func bridgePackageNameOr(origin node.Node, fallback string) string {
	pkg, ok := origin.(*node.Package)
	if !ok || pkg == nil {
		return fallback
	}
	got, ok := bridgeGoNameKey.Get(pkg.Meta())
	if !ok || got == "" {
		return fallback
	}
	return got
}

// bridgePackageImportOr returns the bridge-stamped `go.import`
// from origin or fallback when no stamp is present. Same lookup
// pattern as [bridgePackageNameOr].
func bridgePackageImportOr(origin node.Node, fallback string) string {
	pkg, ok := origin.(*node.Package)
	if !ok || pkg == nil {
		return fallback
	}
	got, ok := bridgeGoImportKey.Get(pkg.Meta())
	if !ok || got == "" {
		return fallback
	}
	return got
}
