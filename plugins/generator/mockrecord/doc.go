// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package mockrecord is the call-recording cross-cutter for
// [plugins/generator/mock]. For every emit struct stamped with
// `mock.iface` (the meta key the mock plugin sets on its output),
// it decorates the mock with per-method call-recording machinery:
// a typed `<MockName><Method>Call` struct, a slice field on the
// mock, and a Prebody contribution that appends to the slice at
// every dispatch.
//
// # Generated shape
//
// Given the mock struct the `mock` plugin produces:
//
//	type StoreMock struct {
//	    OnGet func(ctx context.Context, key string) (Record, error)
//	}
//
//	func (m *StoreMock) Get(ctx context.Context, key string) (_v0 Record, _v1 error) {
//	    if m.OnGet != nil { return m.OnGet(ctx, key) }
//	    return
//	}
//
// `mockrecord` contributes — via slot extensions, not by replacing
// the mock plugin's output:
//
//	type StoreMockGetCall struct {
//	    Ctx context.Context
//	    Key string
//	}
//
//	// appended to StoreMock via FieldsSlot:
//	//   GetCalls []StoreMockGetCall
//
//	// appended to StoreMock.Get's Prebody:
//	//   m.GetCalls = append(m.GetCalls, StoreMockGetCall{Ctx: ctx, Key: key})
//
// The Prebody runs before the mock's nil-check, so a call is
// recorded whether or not an override is installed. Tests assert
// on `m.GetCalls` directly — typed access, no reflection, no
// string-keyed method-name indirection.
//
// # Discovery
//
// `mockrecord` does not depend on the mock plugin's internal
// options or naming conventions — it locates its inputs through
// the [mock.MetaIface] meta key stamped on every mock struct.
// Any alternative mock generator that stamps the same key plugs
// in transparently.
//
// # Routing
//
// `mockrecord` anchors every emitted call struct on the same
// source node the mock anchored on (read from the mock emit
// struct's [emit.BaseEmit.Origin]). Sharing an origin means
// sharing routing: the call structs and the recorder's slot
// contributions follow the mock into whatever package the
// `+gen:out` / `+gen:mock out=...` / project-config layers
// resolve to.
//
// # Extension surface
//
// Downstream plugins that depend on the recorder declare
// `Requires(mockrecord.Capability)` to order after it. The
// recording fields and Prebody statements are themselves
// available for further extension — every contribution carries
// `SetBy = "mockrecord"` so positional inserts via
// [emit.Slot.InsertBefore] / [emit.Slot.InsertAfter] target the
// recorder's contributions precisely.
//
// # Position in the pipeline
//
// `mockrecord` runs in [sdk.GeneratorCrossCutting] and requires
// the `mock` capability — it executes strictly after the mock
// generator populates the emit store. Provides
// `mock-recording` so dependent plugins (assertion helpers,
// replay generators) can order themselves after the recorder.
package mockrecord
