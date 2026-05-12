// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
)

// The protobuf frontend stamps proto-specific facts under the
// `proto.*` namespace. The frontend-provenance marker is the one
// exception: it carries the bare name `frontend` because it is a
// framework-level convention shared across every frontend (the Go
// frontend stamps the same key with value `"golang"`, future
// frontends with their own plugin names). Cross-frontend consumers
// — the protogo bridge annotator, the cross-namespace audit step —
// pivot on the marker to scope language-namespaced meta to the
// correct source.
//
// Per-frontend keys go through [meta.NewKey] (single-owner singletons
// that panic on duplicate registration — the desired behaviour for
// the namespaced `proto.*` surface). The cross-frontend marker key
// goes through [meta.EnsureKey] so multiple frontends can declare
// it independently without an init-order coupling: whichever
// frontend's package initialises first registers the key; the
// later-initialising frontend's [meta.EnsureKey] returns the same
// singleton instead of panicking.
//
// Dynamic per-option keys (proto custom options carrying their full
// dotted name) also go through [meta.EnsureKey] via [OptionMetaKey]
// so each option full-name resolves to a single canonical
// [meta.Key] across the run.
var (
	// MetaFrontend stamps the producing frontend's plugin name on
	// every [node.Package] entry the frontend emits. The value is
	// the string `"protobuf"` for proto-derived packages; the Go
	// frontend stamps the matching `"golang"` on its packages. The
	// marker is the cross-frontend scope mechanism: bridge
	// annotators filter their walk to packages carrying the
	// expected marker value, and the cross-namespace audit
	// assertion proves no `<lang>.*` meta leaks onto sources from
	// other frontends.
	MetaFrontend = meta.EnsureKey(
		"frontend",
		meta.StringParser,
	) //nolint:gochecknoglobals // cross-frontend registry-singleton key
)

// MetaOptionPrefix is the namespace under which proto custom-option
// entries are stamped on host meta bags. For a custom option named
// `my.package.feature_flag`, the converter stamps
// `proto.option.my.package.feature_flag=<typed value>` per the
// option-channel convention. Convenience aliases for high-traffic
// standard options live under sibling `proto.*` keys (e.g.
// `proto.deprecated`, `proto.json_name`); the raw form under
// MetaOptionPrefix is always present alongside the alias.
const MetaOptionPrefix = "proto.option."

// OptionMetaKey returns the typed [meta.Key] for the proto custom
// option whose full-name is name. The returned key lives under
// [MetaOptionPrefix]; the supplied parser determines the typed Go
// value the resolved option carries — signed-int width scalars,
// unsigned-int width scalars, floats, bool, string, bytes,
// enum-name strings, message-shaped payloads, and their composite
// forms. Dynamic names go through [meta.EnsureKey] so the registry
// retains exactly one canonical key per option full-name even when
// the converter visits the same option from multiple call sites.
func OptionMetaKey[T any](name string, parser meta.Parser[T]) meta.Key[T] {
	return meta.EnsureKey(MetaOptionPrefix+name, parser)
}

// optionKeyName returns the canonical option-channel key fragment
// for field. Custom extensions stamp under their proto FQN;
// standard options on google.protobuf.FileOptions and friends
// stamp under the short well-known name (`go_package`,
// `deprecated`, `allow_alias`, …).
func optionKeyName(field protoreflect.FieldDescriptor) string {
	if field.IsExtension() {
		return string(field.FullName())
	}
	return string(field.Name())
}

// stampFrontendMarker records the cross-frontend provenance marker
// on pkg's meta bag. Every produced package carries this stamp;
// bridge annotators and the cross-namespace audit step pivot on it
// to scope their walks to the correct frontend's sources.
func stampFrontendMarker(pkg *node.Package) {
	MetaFrontend.Set(pkg.Meta(), FrontendName, FrontendName)
}
