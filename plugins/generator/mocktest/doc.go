// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package mocktest is the companion test generator for
// [plugins/generator/mock]. For every emit struct stamped with
// `mock.iface` (the meta key the mock plugin sets on its output),
// it emits one `Test<MockName>` function whose body nests two
// levels of subtests — one `t.Run` per dispatch method, each
// containing two `t.Run` branches that cover the nil-check
// dispatch the mock generates.
//
// The nested-subtest shape lets `go test -run` filter at any
// level — by mock (`go test -run TestStoreMock`), method
// (`TestStoreMock/Get`), or branch
// (`TestStoreMock/Get/nil dispatch`) — without bespoke flag
// plumbing.
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
// `mocktest` emits:
//
//	func TestStoreMock(t *testing.T) {
//	    t.Parallel()
//	    t.Run("Get", func(t *testing.T) {
//	        t.Parallel()
//	        t.Run("nil dispatch", func(t *testing.T) {
//	            t.Parallel()
//	            var m StoreMock
//	            var ctx context.Context
//	            var key string
//	            m.Get(ctx, key)
//	        })
//	        t.Run("override dispatch", func(t *testing.T) {
//	            t.Parallel()
//	            called := false
//	            m := StoreMock{OnGet: func(context.Context, string) (Record, error) {
//	                called = true
//	                var _v0 Record
//	                var _v1 error
//	                return _v0, _v1
//	            }}
//	            var ctx context.Context
//	            var key string
//	            m.Get(ctx, key)
//	            if !called {
//	                t.Fatal("OnGet override was not invoked")
//	            }
//	        })
//	    })
//	}
//
// The named-return / naked-return pattern keeps the override closure
// agnostic of return-type construction — no per-type zero-value
// expression has to be synthesised.
//
// # Extension surface
//
// Other plugins extend an emitted test function through the
// standard [emit.Function] slots — `mocktest` owns the canonical
// nested subtests and nothing else:
//
//   - `Function.Prebody()` — statements run before `t.Parallel()`
//     (rare; useful for global setup hooks).
//   - `Function.Postbody()` — statements run after the canonical
//     per-method subtests. The idiomatic extension point: append
//     further `t.Run(...)` blocks (property tests, fuzz cases,
//     recorded replay) without re-implementing mocktest.
//
// `mocktest` keys its contributions with `SetBy = "mocktest"` so
// downstream plugins can position relative to them via
// [emit.Slot.InsertBefore] / [emit.Slot.InsertAfter] when fine
// control is required.
//
// # Discovery
//
// `mocktest` does not depend on the mock plugin's internal options
// or naming conventions. It locates its inputs through two meta
// keys the mock plugin stamps:
//
//   - [mock.MetaIface] on the emit struct — gates which structs are
//     mocks worth processing.
//   - [mock.MetaField] on each emit method — carries the override
//     field name (`OnGet`) so the override-dispatch subtest can
//     construct the composite literal without re-deriving the
//     plugin's `FieldPrefix` / `FieldSuffix` options.
//
// This keeps mocktest decoupled from the producing plugin: any
// alternative mock generator that stamps the same meta keys plugs
// in transparently.
//
// # Position in the pipeline
//
// `mocktest` runs in [sdk.GeneratorCrossCutting] and requires the
// `mock` capability — it executes strictly after the mock
// generator has populated the emit store and before downstream
// cross-cutters that further decorate the mock.
package mocktest
