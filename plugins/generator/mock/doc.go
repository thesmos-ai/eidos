// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package mock is the production mock generator. For every
// source-side interface annotated with `+gen:mock`, it emits a
// `<Type><Suffix>` struct backed by one `<FieldPrefix><Method>`
// func-valued field per interface method, plus an implementing
// method that dispatches to that field with a nil-check
// fall-through to zero values.
//
// # Generated shape
//
// Given:
//
//	//+gen:mock
//	type Store interface {
//	    Get(ctx context.Context, key string) (Record, error)
//	}
//
// The plugin emits:
//
//	type StoreMock struct {
//	    OnGet func(ctx context.Context, key string) (Record, error)
//	}
//
//	func (m *StoreMock) Get(ctx context.Context, key string) (_v0 Record, _v1 error) {
//	    if m.OnGet != nil {
//	        return m.OnGet(ctx, key)
//	    }
//	    return
//	}
//
// Named anonymous returns produce a naked `return` that yields
// the zero value of every result type — no per-return-type
// zero-value-expression construction is required.
//
// # Extension surface
//
// Every cross-cutting concern attaches to the standard slot
// surface of the emitted entities. The mock plugin owns nothing
// beyond the struct + its methods; everything else composes via
// slots that downstream plugins write to:
//
//   - `Struct.FieldsSlot()` — append extra fields (e.g.
//     `gets []GetCall` for a recording plugin).
//   - `Struct.MethodsSlot()` — append helper methods (e.g.
//     `Reset()`, `Calls()`).
//   - `Method.Prebody()` — statements that run before the
//     dispatch nil-check (e.g. recording the call, applying
//     faults).
//   - `Method.Postbody()` — statements that run after the
//     dispatch returns.
//
// Provenance IDs the plugin attaches to its own contributions
// follow `mock.<role>` so downstream plugins position relative
// to specific contributions via [emit.Slot.InsertBefore] /
// [emit.Slot.InsertAfter].
//
// # Composition example
//
// A testkit-style recording plugin layered on top:
//
//	for _, m := range ctx.Reader.EmitMethods().Slice() {
//	    if !belongsToMock(m) { continue }
//	    m.Prebody().Append(
//	        emit.NewExprStmt(emit.NewCall(
//	            emit.NewField(emit.NewIdent("m"), "record"),
//	        )),
//	        emit.Provenance{Plugin: "mock-recording", ID: "mock.recording"},
//	    )
//	}
//
// Other plugins (fault injection, bench mode, tracing) attach to
// the same slots without coordinating with each other; the
// framework's capability topo sort orders their contributions by
// `Provides` / `Requires` declarations.
//
// # Position in the pipeline
//
// The plugin runs in the [sdk.GeneratorComposition] bucket: it
// emits onto the source-side interface only, requiring no prior
// generator output. Cross-cutting plugins that decorate the
// mock register in [sdk.GeneratorCrossCutting] so they see the
// composed mock surface.
package mock
