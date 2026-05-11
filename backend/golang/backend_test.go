// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/testpipe"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// TestBackend_Name covers the stable plugin identifier.
func TestBackend_Name(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented constant", func(t *testing.T) {
		t.Parallel()
		if got := mustNew(t).Name(); got != golang.Name {
			t.Fatalf("Name = %q, want %q", got, golang.Name)
		}
	})

	t.Run("Name is namespaced for collision avoidance", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(golang.Name, ".") {
			t.Fatalf("Name %q should be namespaced (contain '.')", golang.Name)
		}
	})
}

// TestBackend_Language covers the target-language identifier.
func TestBackend_Language(t *testing.T) {
	t.Parallel()

	t.Run("Language returns the documented constant", func(t *testing.T) {
		t.Parallel()
		if got := mustNew(t).Language(); got != golang.Language {
			t.Fatalf("Language = %q, want %q", got, golang.Language)
		}
	})
}

// TestBackend_Render covers the per-Target render orchestration.
func TestBackend_Render(t *testing.T) {
	t.Parallel()

	t.Run("empty store produces no sink writes and no diagnostics", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render on empty store: %v", err)
		}
		if mem.Len() != 0 {
			t.Fatalf("expected no sink writes; got %d", mem.Len())
		}
		if len(d.Diagnostics()) != 0 {
			t.Fatalf("expected no diagnostics; got %+v", d.Diagnostics())
		}
	})

	t.Run("writes one gofmt-clean file per distinct Target", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		t1 := emit.Target{Dir: "out/users", Filename: "user.go", Package: "users"}
		t2 := emit.Target{Dir: "out/orders", Filename: "order.go", Package: "orders"}
		addEmitPackage(t, ctx, emitPackage("users", emitStructWithFields(
			"users", "User", t1,
			fieldSpec{name: "ID", builtin: "int"},
			fieldSpec{name: "Name", builtin: "string"},
		)))
		addEmitPackage(t, ctx, emitPackage("orders", emitStructWithFields(
			"orders", "Order", t2,
			fieldSpec{name: "Total", builtin: "int"},
		)))
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if d.HasErrors() {
			t.Fatalf("unexpected error diagnostics: %+v", d.Diagnostics())
		}
		if mem.Len() != 2 {
			t.Fatalf("expected 2 sink writes; got %d (files=%v)", mem.Len(), mem.Files())
		}
		// Spot-check each file's package line and decl line.
		for _, c := range []struct {
			target  emit.Target
			pkgLine string
			decl    string
		}{
			{t1, "package users", "type User struct {"},
			{t2, "package orders", "type Order struct {"},
		} {
			body, ok := mem.Get(c.target)
			if !ok {
				t.Fatalf("no output for %v", c.target)
			}
			if !strings.Contains(string(body), c.pkgLine) {
				t.Fatalf("%v: body should contain %q; got:\n%s", c.target, c.pkgLine, body)
			}
			if !strings.Contains(string(body), c.decl) {
				t.Fatalf("%v: body should contain %q; got:\n%s", c.target, c.decl, body)
			}
		}
	})

	t.Run("zero-valued Target never reaches the sink", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		// Zero Target — store's by-target index drops it on insert, so
		// nothing reaches the render loop. Verifies the upstream
		// filter via the public render path.
		addEmitPackage(t, ctx, emitPackage(
			"x",
			emitStructWithFields("x", "X", emit.Target{}, fieldSpec{name: "F", builtin: "int"}),
		))
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if mem.Len() != 0 {
			t.Fatalf("zero-target struct should not produce sink output; got %d files", mem.Len())
		}
		if d.HasErrors() {
			t.Fatalf("unexpected error diagnostics: %+v", d.Diagnostics())
		}
	})

	t.Run("sink failure surfaces as a wrapped error from Render", func(t *testing.T) {
		t.Parallel()
		ctx, _, _ := newBackendContext(t)
		ctx.Sink = &failingSink{}
		target := emit.Target{Dir: "out", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, emitPackage("x", emitStructWithFields(
			"x", "X", target,
			fieldSpec{name: "F", builtin: "int"},
		)))
		err := mustNew(t).Render(ctx)
		if err == nil {
			t.Fatalf("expected an error when sink fails")
		}
		if !errors.Is(err, errSinkBoom) {
			t.Fatalf("err should wrap errSinkBoom; got %v", err)
		}
		msg := err.Error()
		if !strings.Contains(msg, golang.Name) {
			t.Fatalf("error %q should mention backend Name %q", msg, golang.Name)
		}
		if !strings.Contains(msg, target.JoinPath()) {
			t.Fatalf("error %q should mention target path %q", msg, target.JoinPath())
		}
	})
}

// TestBackend_Golden pins canonical output for each
// Phase-C-shipped fixture so byte-level drift in templates,
// funcmap, format.Source, or goimports is caught at PR time.
// Each subtest covers a representative shape from the Phase C
// acceptance criteria.
func TestBackend_Golden(t *testing.T) {
	t.Parallel()

	t.Run("struct_simple — no imports", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		addEmitPackage(t, ctx, emitPackage("users", emitStructWithFields(
			"users", "User", target,
			fieldSpec{name: "ID", builtin: "int"},
			fieldSpec{name: "Name", builtin: "string"},
		)))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_simple.go.golden"))
	})

	t.Run("struct_stdlib_import — context.Context field", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		addEmitPackage(t, ctx, emitPackage("users", &emit.Struct{
			Name: "Request", Package: "users", Target: target,
			Fields: []*emit.Field{{Name: "Ctx", Type: emit.External("context", "Context")}},
		}))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_stdlib_import.go.golden"))
	})

	t.Run("struct_external_import — third-party type", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		addEmitPackage(t, ctx, emitPackage("users", &emit.Struct{
			Name: "Wrapper", Package: "users", Target: target,
			Fields: []*emit.Field{{Name: "Inner", Type: emit.External("github.com/example/lib", "Item")}},
		}))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_external_import.go.golden"))
	})

	t.Run("struct_multi_import — stdlib + external regrouped", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		addEmitPackage(t, ctx, emitPackage("users", &emit.Struct{
			Name: "Request", Package: "users", Target: target,
			Fields: []*emit.Field{
				{Name: "Ctx", Type: emit.External("context", "Context")},
				{Name: "Err", Type: emit.External("errors", "Is")},
				{Name: "Item", Type: emit.External("github.com/example/lib", "Item")},
			},
		}))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_multi_import.go.golden"))
	})

	t.Run("struct_with_docs — DocLines render as // above the decl", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		addEmitPackage(t, ctx, emitPackage("users", &emit.Struct{
			BaseEmit: emit.BaseEmit{
				DocLines: []string{
					"User is the canonical user record.",
					"",
					"Fields hold the immutable identifier and the display",
					"name used in UI surfaces.",
				},
			},
			Name: "User", Package: "users", Target: target,
			Fields: []*emit.Field{
				{Name: "ID", Type: emit.Builtin("int")},
				{Name: "Name", Type: emit.Builtin("string")},
			},
		}))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_with_docs.go.golden"))
	})

	t.Run("struct_with_directive_in_docs — directive lines ride DocLines, rendered verbatim", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		// Generators put `//nolint:foo` and similar suppressions at
		// the END of DocLines per Go convention. `renderDocs` detects
		// the leading "//" and renders the line verbatim; regular
		// doc lines get the "// " prefix applied.
		addEmitPackage(t, ctx, emitPackage("users", &emit.Struct{
			BaseEmit: emit.BaseEmit{
				DocLines: []string{
					"Legacy is kept around for backwards compatibility.",
					"//nolint:revive",
				},
			},
			Name: "Legacy", Package: "users", Target: target,
			Fields: []*emit.Field{{Name: "ID", Type: emit.Builtin("int")}},
		}))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_with_directive_in_docs.go.golden"))
	})

	t.Run("package_with_docs — emit.Package.DocLines surface above package decl", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		// Package-level docs ride on emit.Package.DocLines; the
		// backend applies them as the file's package doc when no
		// per-Target emit.File overrides them.
		pkg := &emit.Package{
			BaseEmit: emit.BaseEmit{
				DocLines: []string{
					"Package users models the canonical user record and",
					"the operations performed on it across the platform.",
				},
			},
			Name: "users", Path: "users",
			Structs: []*emit.Struct{
				{
					Name: "User", Package: "users", Target: target,
					Fields: []*emit.Field{{Name: "ID", Type: emit.Builtin("int")}},
				},
			},
		}
		addEmitPackage(t, ctx, pkg)
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "package_with_docs.go.golden"))
	})

	t.Run("struct_with_field_annotations — field docs, tags, line comments", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		addEmitPackage(t, ctx, emitPackage("users", &emit.Struct{
			BaseEmit: emit.BaseEmit{
				DocLines: []string{"User is the canonical user record."},
			},
			Name: "User", Package: "users", Target: target,
			Fields: []*emit.Field{
				{
					BaseEmit: emit.BaseEmit{
						DocLines: []string{
							"ID is the immutable primary key.",
							"Stored as the database row identifier.",
						},
					},
					Name: "ID",
					Type: emit.Builtin("int"),
					Tag:  `json:"id"`,
				},
				{
					BaseEmit: emit.BaseEmit{
						DocLines: []string{"Name is the display name shown in the UI."},
					},
					Name:        "Name",
					Type:        emit.Builtin("string"),
					Tag:         `json:"name"`,
					LineComment: "max 64 chars per product spec",
				},
				{
					Name:        "Internal",
					Type:        emit.Builtin("bool"),
					LineComment: "set by middleware; not exposed externally",
				},
			},
		}))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_with_field_annotations.go.golden"))
	})

	t.Run("struct_alias_collision — suffix-2 deterministic alias", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, emitPackage("x", &emit.Struct{
			Name: "X", Package: "x", Target: target,
			Fields: []*emit.Field{
				{Name: "A", Type: emit.External("github.com/example/users", "User")},
				{Name: "B", Type: emit.External("github.com/other/users", "User")},
			},
		}))
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_alias_collision.go.golden"))
	})
}

// assertRenderSucceeds drives the backend over ctx, asserting no
// errors and a non-empty sink output for target, then returns the
// rendered bytes for golden comparison. Centralised so each golden
// subtest stays at the "build fixture, assert golden" altitude.
func assertRenderSucceeds(
	t *testing.T,
	ctx *plugin.BackendContext,
	mem *sink.Memory,
	d *diag.Sink,
	target emit.Target,
) []byte {
	t.Helper()
	if err := mustNew(t).Render(ctx); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if d.HasErrors() {
		t.Fatalf("unexpected error diagnostics: %+v", d.Diagnostics())
	}
	body, ok := mem.Get(target)
	if !ok {
		t.Fatalf("no output for %v", target)
	}
	return body
}
