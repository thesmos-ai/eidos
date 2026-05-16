// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package shape is the framework for classifying callable
// signatures (free functions and methods) into named shapes that
// downstream generators consume. The package owns the universal
// contract — three typed meta keys, the `+gen:shape` directive,
// the umbrella [Plugin], and the lazy Go-language helpers — but
// knows nothing about any individual shape. Each shape (Reader,
// Writer, Lifecycle, …) lives in its own sub-package alongside
// this one and exports a [Detector] the consumer composes in.
//
// # The contract
//
// A shape stamp is three meta keys on a callable's bag:
//
//   - `shape`            — the canonical name (string, e.g. "reader")
//   - `shape.key_type`   — qualified type of the key/input parameter, when applicable
//   - `shape.value_type` — qualified type of the value/output, when applicable
//
// The triple is the universal contract — same across every source
// language. A Go `func(ctx, K) (V, error)` and a hypothetical
// Rust `fn(&self, K) -> Result<V, E>` both surface as
// `shape = "reader"`; downstream consumers branch on the stamp
// without caring about the source frontend.
//
// # Multi-language detection
//
// Detection logic differs per source language. A shape sub-package
// exports a [Detector] whose [Detector.Detect] map carries one
// [DetectFunc] per supported frontend:
//
//	func Detector() shape.Detector {
//	    return shape.Detector{
//	        Name: "reader",
//	        Detect: map[string]shape.DetectFunc{
//	            "golang":   detectGolang,
//	            "protobuf": detectProtobuf,
//	            "rust":     detectRust,
//	        },
//	    }
//	}
//
// The umbrella plugin reads each package's `frontend` marker meta
// and dispatches to the matching [DetectFunc]. Sources from
// frontends without a registered entry are silently skipped —
// permissive by construction.
//
// # Composition
//
// The umbrella [Plugin] takes every shape's [Detector] at
// construction time and runs them in registration order on every
// callable in the store:
//
//	pipe.Use(shape.New(
//	    reader.Detector(),
//	    writer.Detector(),
//	    lifecycle.Detector(),
//	))
//
// One plugin owns the `+gen:shape` directive schema, one walks
// the node graph (via the framework's [plugin.Walk] hook
// dispatcher), one applies the override-then-detect cascade.
// There is no separate directive plugin to register and no risk
// of duplicate-directive registration when the consumer wants
// more than one shape.
//
// # User overrides
//
// A `+gen:shape <name>` directive on a callable wins over every
// detector. The umbrella plugin checks the directive list first
// per callable; on a hit, the matching meta keys are stamped and
// the detector cascade is skipped via the "already stamped" guard.
//
// # Traversal
//
// The plugin implements the framework's per-callable visitor
// hooks ([plugin.MethodHook] and [plugin.FunctionHook]) so the
// pipeline's single annotator walk covers every callable; there
// is no plugin-local tree traversal.
//
// # Where the boundary sits
//
// The shape library handles every concern except the shape's own
// detection logic and name:
//
//   - It owns the `+gen:shape` directive schema and the override
//     pre-stamp pass.
//   - It receives every callable through the framework's
//     [plugin.Walk] hook dispatcher.
//   - It looks up the per-frontend [DetectFunc] for the package's
//     source language.
//   - It applies the already-stamped skip.
//   - It stamps the three meta keys on a positive [Match].
//
// Per-shape packages contribute only the [DetectFunc] bodies and
// the canonical name they stamp. The shape vocabulary is the
// union of registered shape packages; nothing in this library
// enumerates the world.
package shape
