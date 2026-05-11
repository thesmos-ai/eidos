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

// TestBackend_GoldenStructSimple pins the canonical struct output
// against a checked-in golden file so byte-level drift in the
// template, the funcmap, or go/format.Source is caught at PR time.
func TestBackend_GoldenStructSimple(t *testing.T) {
	t.Parallel()

	t.Run("simple struct fixture matches checked-in golden", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		addEmitPackage(t, ctx, emitPackage("users", emitStructWithFields(
			"users", "User", target,
			fieldSpec{name: "ID", builtin: "int"},
			fieldSpec{name: "Name", builtin: "string"},
		)))
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
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_simple.go.golden"))
	})
}
