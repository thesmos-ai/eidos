// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package sdk is the plugin-author façade. It re-exports the
// contract-shaped surface from the framework's layered packages —
// [plugin], [priority], [core/directive], and [core/kind] — under
// one import so a new plugin's `import` block stays compact.
//
// A typical plugin imports:
//
//	import (
//	    "go.thesmos.sh/eidos/sdk"          // role + capability contracts
//	    "go.thesmos.sh/eidos/core/opt"     // typed options (plugins that have any)
//	    "go.thesmos.sh/eidos/node"         // source-side store (annotators / generators)
//	    "go.thesmos.sh/eidos/emit"         // emit-side store (generators / backends)
//	    "go.thesmos.sh/eidos/emit/builder" // emit construction helpers (generators)
//	)
//
// Without the façade the same plugin needs `core/directive`,
// `priority`, `plugin`, `core/kind`, plus the four high-volume
// packages above — eight imports before the plugin's own code
// starts. sdk halves that to four.
//
// # What's in scope
//
// Every contract-shaped concern: the role and capability
// interfaces, per-phase contexts, annotator visitor hooks,
// priority buckets, directive schema declaration + validation,
// and the structural kind discriminator. See the per-source
// files in this package for the grouped surface:
//
//   - [plugin.go]: roles, capabilities, contexts, hooks
//   - [priority.go]: Priority type + canonical buckets
//   - [directive.go]: schemas, parsing, validation, registry
//   - [kind.go]: Kind type
//
// # What's intentionally out of scope
//
// The high-volume packages — [node], [emit], [emit/builder],
// [core/opt], [core/meta] — stay as separate imports. Flattening
// them here would balloon this package's surface to ~700 symbols
// and shadow common Go names (`Field`, `Schema`, `Type`, `Walk`,
// `New`) with multiple conflicting meanings. Plugin authors who
// want the convenience of one import per concern get four
// imports total; the bulk volume is in the emit-construction
// code, where the [emit] and [emit/builder] packages already own
// a coherent namespace.
package sdk
