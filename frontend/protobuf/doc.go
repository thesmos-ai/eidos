// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package protobuf is the proto3 frontend for eidos. It loads
// `.proto` sources via [github.com/bufbuild/protocompile] and
// converts the resulting descriptor set into the language-agnostic
// [node] model — messages, enums, services, oneofs, maps, repeated
// fields, nested types, well-known references, custom options,
// reserved ranges, and comments all surface as the corresponding
// node kinds with proto-specific facts stamped under the `proto.*`
// meta namespace.
//
// # Scope
//
// The frontend supports `syntax = "proto3";` only. Sources whose
// first declaration is `edition = "..."` (the proto editions
// migration path, edition 2023 and later) surface a positioned
// `diag.Error` and the file is skipped; remaining files in the
// same load continue to parse. proto2 is not supported.
//
// # Pipeline integration
//
// The frontend implements [plugin.Frontend] and is registered on a
// [pipeline.Builder] via [pipeline.Builder.WithFrontend]:
//
//	pipeline.New().
//	    WithFrontend(protobuf.New()).
//	    WithBackend(backend_golang.New()).
//	    Build()
//
// Load is invoked once per [plugin.FrontendContext.Pattern]. Each
// pattern is interpreted as a proto-source root or import path;
// protocompile's `Compiler` resolves transitive imports against
// the configured `import_paths` plus the search root.
//
// # Output vocabulary
//
// The frontend produces source-side [node.Package] entries keyed by
// the proto package qualifier on `Path` and the last dotted
// segment on `Name`. Type-reference values in `TypeRef.Name` carry
// the proto source surface verbatim (e.g. `int32`, `sint32`,
// `fixed64`, `float`, `bytes`); cross-language translation is the
// `protogo` bridge annotator's responsibility (see
// `reference/protogo`), not the frontend's.
//
// # Bridge annotator
//
// Proto→Go pipelines pair the frontend with the `protogo`
// annotator. protogo stamps `go.*` meta on every proto-derived
// type-ref-bearing host (field, method param, method return);
// the Go backend's render-site rules prefer the `go.*` meta over
// the underlying proto-native name. Pipelines targeting non-Go
// outputs swap the bridge for the matching language pair (a
// future `protorust` / `prototypescript`) or skip the bridge
// entirely for proto-native targets.
package protobuf
