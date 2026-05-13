// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package protogo is the proto→Go bridge annotator. It walks
// proto-derived source nodes (every package whose `frontend` meta
// equals `"protobuf"`) and stamps Go-namespaced translation meta
// so the existing Go backend renders compilable Go without
// learning anything proto-specific.
//
// # Stamps
//
//   - [MetaGoType] on every [node.TypeRef] reachable from a
//     field, method param, or method return. The value is the
//     Go-side rendered form (`int32`, `[]byte`, `*timestamppb.Timestamp`,
//     `[]*foopb.Item`).
//   - [MetaGoName] on every [node.Field] with the Go-idiomatic
//     PascalCase identifier (`user_id` → `UserID`).
//   - [MetaGoName] + [MetaGoImport] on every [node.Package] —
//     `MetaGoImport` reads the proto `go_package` option,
//     `MetaGoName` derives the rendered package clause.
//
// # Idempotency
//
// Pre-stamped Go-namespaced meta wins over the translation
// tables. Users override individual hosts by setting
// `go.type` / `go.name` / `go.import` themselves before the
// bridge runs; the bridge's "absent → stamp" rule preserves the
// override.
//
// # Scope
//
// The bridge filters its iteration to packages stamped with the
// `frontend = "protobuf"` marker. Go-derived packages are
// untouched. The cross-frontend audit step verifies the scope
// guard against a Go+proto pipeline.
//
// # Pipeline integration
//
//	pipeline.New().
//	    WithFrontend(protobuf.New()).
//	    WithAnnotator(protogo.New()).
//	    WithBackend(backend_golang.New()).
//	    Build()
//
// Generators downstream of protogo see the Go-namespaced meta on
// every relevant node; the Go backend's render-site rules read
// the meta first and fall back to the underlying node-level
// name when the bridge hasn't run.
package protogo
