// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo

import "go.thesmos.sh/eidos/core/meta"

// The bridge stamps Go-side facts under the `go.*` namespace —
// the same namespace the Go frontend uses for its own
// language-specific stamps. Consumers (the Go backend's
// render-site rules, downstream generators that want Go-flavoured
// type information) read these keys without dispatching on
// frontend origin.
//
// Each key uses [meta.EnsureKey] so multiple bridges (a future
// `protorust`, `prototypescript`) can declare overlapping keys
// without an init-order coupling. Cross-language consumers that
// need the Go-side facts always read these stable names.
var (
	// MetaGoType stamps the Go-side rendered type expression on
	// a [node.TypeRef]. The value is the full rendered form —
	// `int32` for a scalar, `[]byte` for bytes, `*timestamppb.Timestamp`
	// for a well-known reference, `[]*pkg.Item` for a repeated
	// message. The Go backend's TypeRef render-site reads this
	// key first and falls back to the underlying name verbatim
	// when the bridge hasn't run.
	MetaGoType = meta.EnsureKey(
		"go.type",
		meta.StringParser,
	) //nolint:gochecknoglobals // cross-bridge registry-singleton key

	// MetaGoName stamps the Go-idiomatic identifier for a
	// [node.Field] or [node.Package]. On a Field, the value is
	// the PascalCase form of the proto field name (`user_id` →
	// `UserID`). On a Package, the value is the Go-side package
	// clause (`acmev1`).
	MetaGoName = meta.EnsureKey(
		"go.name",
		meta.StringParser,
	) //nolint:gochecknoglobals // cross-bridge registry-singleton key

	// MetaGoImport stamps the Go import path on a [node.Package]
	// derived from the proto `go_package` option (the
	// `proto.option.go_package` meta key the frontend already
	// stamps). Cross-package references through the rendered Go
	// output resolve through this key's value rather than the
	// proto package qualifier.
	MetaGoImport = meta.EnsureKey(
		"go.import",
		meta.StringParser,
	) //nolint:gochecknoglobals // cross-bridge registry-singleton key
)
