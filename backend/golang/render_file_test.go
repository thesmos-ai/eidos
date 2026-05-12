// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/testpipe"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

// TestFileCompose_SharedTargetTwoPlugins covers the happy-path of
// multi-generator file composition: two plugins each call
// AddPackage with one Struct routed to the same Target. The
// backend composes both decls into a single rendered file. Both
// Structs must appear, in plugin registration order (which
// corresponds to AddPackage call order since the store records
// insertion order for ByTarget).
func TestFileCompose_SharedTargetTwoPlugins(t *testing.T) {
	t.Parallel()

	t.Run("two AddPackage calls share a Target; both decls render", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		ctx.Ordered = []plugin.Plugin{
			stubPluginVersion{name: "users"},
			stubPluginVersion{name: "events"},
		}
		target := emit.Target{Dir: "domain", Filename: "core.go", Package: "domain"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "domain.users", Path: "domain/users",
			Structs: []*emit.Struct{{
				Name: "User", Package: "domain/users", Target: target,
				Fields: []*emit.Field{{Name: "ID", Type: emit.Builtin("int")}},
			}},
		})
		addEmitPackage(t, ctx, &emit.Package{
			Name: "domain.events", Path: "domain/events",
			Structs: []*emit.Struct{{
				Name: "Event", Package: "domain/events", Target: target,
				Fields: []*emit.Field{{Name: "Kind", Type: emit.Builtin("string")}},
			}},
		})
		body := string(assertRenderSucceeds(t, ctx, mem, d, target))
		mustOrderedSubstrings(t, body, "type User struct {", "type Event struct {")
	})
}

// TestFileCompose_InitBlockFromThreePlugins covers init composition
// — three plugins each contribute one statement to the file's
// [emit.File.Init] slot; the backend composes them into a single
// `func init() { … }` block, topo-ordered.
func TestFileCompose_InitBlockFromThreePlugins(t *testing.T) {
	t.Parallel()

	t.Run("three plugins contribute init stmts; single init block emitted", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		ctx.Ordered = []plugin.Plugin{
			stubPluginVersion{name: "logger"},
			stubPluginVersion{name: "metrics"},
			stubPluginVersion{name: "tracer"},
		}
		target := emit.Target{Dir: "boot", Filename: "boot.go", Package: "boot"}
		f, err := ctx.Store.Emit().FileFor(target)
		if err != nil {
			t.Fatalf("FileFor: %v", err)
		}
		appendInit := func(setBy, raw string) {
			if err := f.Init().Append(emit.NewRawStmt(raw), emit.Provenance{SetBy: setBy}); err != nil {
				t.Fatalf("append %s init: %v", setBy, err)
			}
		}
		// Out of topo order — render verifies re-sorting.
		appendInit("tracer", `_ = "tracer-init"`)
		appendInit("metrics", `_ = "metrics-init"`)
		appendInit("logger", `_ = "logger-init"`)
		// FileFor inserted the File with a zero-package; mirror the
		// Target so byTarget routing picks it up.
		f.Package = target.Package
		f.Name = target.Filename
		f.Dir = target.Dir
		body := string(assertRenderSucceeds(t, ctx, mem, d, target))
		mustOrderedSubstrings(
			t, body,
			"func init() {",
			`_ = "logger-init"`,
			`_ = "metrics-init"`,
			`_ = "tracer-init"`,
		)
		if strings.Count(body, "func init()") != 1 {
			t.Fatalf("expected exactly one init block; got:\n%s", body)
		}
	})

	t.Run("empty Init slot emits no init block", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, emitPackage("x", emitStructWithFields(
			"x", "X", target, fieldSpec{name: "F", builtin: "int"},
		)))
		body := string(assertRenderSucceeds(t, ctx, mem, d, target))
		if strings.Contains(body, "func init()") {
			t.Fatalf("file with empty Init slot must not emit func init(); got:\n%s", body)
		}
	})
}

// TestFileCompose_TopBottomSlots covers the file-level slot
// placement — the file's Top slot renders above the free-floating
// decls, Bottom below them, each in plugin-topo order.
func TestFileCompose_TopBottomSlots(t *testing.T) {
	t.Parallel()

	t.Run("top-only target renders header content above package decls", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		ctx.Ordered = []plugin.Plugin{stubPluginVersion{name: "introgen"}}
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		f := bindFile(t, ctx, target)
		topConst := &emit.Constant{
			Name: "Banner", Package: "x", Target: emit.Target{},
			Value: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitString, RawText: "hello"},
		}
		if err := f.Top().Append(topConst, emit.Provenance{SetBy: "introgen"}); err != nil {
			t.Fatalf("append top: %v", err)
		}
		addEmitPackage(t, ctx, emitPackage("x", emitStructWithFields(
			"x", "X", target, fieldSpec{name: "F", builtin: "int"},
		)))
		body := string(assertRenderSucceeds(t, ctx, mem, d, target))
		mustOrderedSubstrings(t, body, `const Banner = "hello"`, "type X struct {")
	})

	t.Run("bottom-only contribution renders below decls", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		ctx.Ordered = []plugin.Plugin{stubPluginVersion{name: "footergen"}}
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		f := bindFile(t, ctx, target)
		bot := &emit.Variable{
			Name: "Sentinel", Package: "x", Target: emit.Target{},
			Init: &emit.Expr{ExprKind: emit.ExprLiteral, LitKind: emit.LitBool, RawText: "true"},
		}
		if err := f.Bottom().Append(bot, emit.Provenance{SetBy: "footergen"}); err != nil {
			t.Fatalf("append bottom: %v", err)
		}
		addEmitPackage(t, ctx, emitPackage("x", emitStructWithFields(
			"x", "X", target, fieldSpec{name: "F", builtin: "int"},
		)))
		body := string(assertRenderSucceeds(t, ctx, mem, d, target))
		mustOrderedSubstrings(t, body, "type X struct {", "var Sentinel = true")
	})
}

// TestFileCompose_EmptyTargetFilter covers the empty-target rule:
// a Target with no decls, no File slots populated produces no
// sink output. Confirms the filter applies through the public
// render path.
func TestFileCompose_EmptyTargetFilter(t *testing.T) {
	t.Parallel()

	t.Run("File with no decls and empty slots produces no output", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		// Create the File entity but contribute nothing to it; the
		// File itself routes through byTarget but carries no decls
		// or slot items, so the empty-target filter fires.
		bindFile(t, ctx, target)
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if d.HasErrors() {
			t.Fatalf("unexpected error diagnostics: %+v", d.Diagnostics())
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("empty Target must not produce a sink write")
		}
	})

	t.Run("ErrEmptyTarget is exported and satisfies errors.Is", func(t *testing.T) {
		t.Parallel()
		if golang.ErrEmptyTarget == nil {
			t.Fatalf("ErrEmptyTarget must be exported and non-nil")
		}
		if !errors.Is(golang.ErrEmptyTarget, golang.ErrEmptyTarget) {
			t.Fatalf("ErrEmptyTarget must satisfy errors.Is reflexivity")
		}
	})
}

// TestFileCompose_DuplicateQNameAcrossPlugins covers the cross-
// plugin duplicate-entity surface: two plugins emit decls with the
// same QName routed to the same Target. Today's store-level dedup
// catches the collision at AddPackage time with
// [store.ErrDuplicateQName]; this test pins that surface so the
// duplicate-entity discipline is observable from the public
// emit-view API.
func TestFileCompose_DuplicateQNameAcrossPlugins(t *testing.T) {
	t.Parallel()

	t.Run("two AddPackage calls for the same struct QName collide", func(t *testing.T) {
		t.Parallel()
		ctx, _, _ := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, emitPackage("x", emitStructWithFields(
			"x", "Duplicate", target, fieldSpec{name: "F", builtin: "int"},
		)))
		err := ctx.Store.Emit().AddPackage(emitPackage("x", emitStructWithFields(
			"x", "Duplicate", target, fieldSpec{name: "G", builtin: "string"},
		)))
		if !errors.Is(err, store.ErrDuplicateQName) {
			t.Fatalf("expected ErrDuplicateQName from second AddPackage; got %v", err)
		}
	})
}

// TestFileCompose_ImportsSlotRegistersBeforeBody covers the pre-
// render imports pass: imports contributed via
// [emit.File.ImportsSlot] register with the writer's
// [writer.ImportSet] before any template fires, so the final
// import block carries the plugin-staged path even when no decl
// references it.
func TestFileCompose_ImportsSlotRegistersBeforeBody(t *testing.T) {
	t.Parallel()

	t.Run("plugin-staged blank import surfaces in the rendered import block", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		ctx.Ordered = []plugin.Plugin{stubPluginVersion{name: "sideeffectgen"}}
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		f := bindFile(t, ctx, target)
		// Side-effect-only imports (Go's `import _ "…"` form) are
		// the canonical reason ImportsSlot exists — they reference
		// no symbol, so the body has no `imp` call that would
		// otherwise pull them in.
		if err := f.ImportsSlot().Append(
			&emit.Import{Path: "embed", Alias: "_"},
			emit.Provenance{SetBy: "sideeffectgen"},
		); err != nil {
			t.Fatalf("append import: %v", err)
		}
		addEmitPackage(t, ctx, emitPackage("x", emitStructWithFields(
			"x", "X", target, fieldSpec{name: "F", builtin: "int"},
		)))
		body := string(assertRenderSucceeds(t, ctx, mem, d, target))
		if !strings.Contains(body, `_ "embed"`) {
			t.Fatalf("expected staged blank import `_ \"embed\"` in body; got:\n%s", body)
		}
	})
}

// TestFileCompose_Goldens pins canonical multi-plugin output for
// the two flagship file-composition scenarios — a shared Target
// with two plugin contributions and a three-plugin init block —
// so byte-level drift in file composition is caught at PR time.
func TestFileCompose_Goldens(t *testing.T) {
	t.Parallel()

	t.Run("file_compose_shared_target — two plugins, one file", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		ctx.Ordered = []plugin.Plugin{
			stubPluginVersion{name: "users"},
			stubPluginVersion{name: "events"},
		}
		target := emit.Target{Dir: "domain", Filename: "core.go", Package: "domain"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "domain.users", Path: "domain/users",
			Structs: []*emit.Struct{{
				BaseEmit: emit.BaseEmit{DocLines: []string{"User is the canonical user record."}},
				Name:     "User", Package: "domain/users", Target: target,
				Fields: []*emit.Field{{Name: "ID", Type: emit.Builtin("int")}},
			}},
		})
		addEmitPackage(t, ctx, &emit.Package{
			Name: "domain.events", Path: "domain/events",
			Structs: []*emit.Struct{{
				BaseEmit: emit.BaseEmit{DocLines: []string{"Event is one entry on the timeline."}},
				Name:     "Event", Package: "domain/events", Target: target,
				Fields: []*emit.Field{{Name: "Kind", Type: emit.Builtin("string")}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "file_compose_shared_target.go.golden"))
	})

	t.Run("file_compose_init_block — three-plugin init composition", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		ctx.Ordered = []plugin.Plugin{
			stubPluginVersion{name: "logger"},
			stubPluginVersion{name: "metrics"},
			stubPluginVersion{name: "tracer"},
		}
		target := emit.Target{Dir: "boot", Filename: "boot.go", Package: "boot"}
		f := bindFile(t, ctx, target)
		appendInit := func(setBy, raw string) {
			if err := f.Init().Append(emit.NewRawStmt(raw), emit.Provenance{SetBy: setBy}); err != nil {
				t.Fatalf("append %s init: %v", setBy, err)
			}
		}
		appendInit("tracer", `_ = "tracer-init"`)
		appendInit("metrics", `_ = "metrics-init"`)
		appendInit("logger", `_ = "logger-init"`)
		addEmitPackage(t, ctx, emitPackage("boot", emitStructWithFields(
			"boot", "Boot", target, fieldSpec{name: "Ready", builtin: "bool"},
		)))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "file_compose_init_block.go.golden"))
	})
}
