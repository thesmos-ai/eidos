// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/emit"
)

// TestErrTemplateMissing covers the sentinel surfaced when the
// render funcmap can't find a template for an entity's Kind. The
// sentinel itself is small but used by every render-dispatch
// branch — pinning its exported shape catches accidental rename
// or removal.
func TestErrTemplateMissing(t *testing.T) {
	t.Parallel()

	t.Run("ErrTemplateMissing is exported and satisfies errors.Is", func(t *testing.T) {
		t.Parallel()
		if golang.ErrTemplateMissing == nil {
			t.Fatalf("ErrTemplateMissing must be exported and non-nil")
		}
		if !errors.Is(golang.ErrTemplateMissing, golang.ErrTemplateMissing) {
			t.Fatalf("ErrTemplateMissing must satisfy errors.Is reflexivity")
		}
	})
}

// TestNewRenderState_LifecycleViaPublicPath verifies the
// clone-per-Target pattern works through the public render path:
// constructing a Backend and calling Render successfully exercises
// newRenderState's clone + funcmap binding once per Target. Any
// regression in the clone mechanism (e.g., shared mutable state
// across renders) would surface as cross-target leakage during a
// multi-target render.
func TestNewRenderState_LifecycleViaPublicPath(t *testing.T) {
	t.Parallel()

	t.Run("renders two distinct targets without cross-target leakage", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		t1 := emit.Target{Dir: "a", Filename: "foo.go", Package: "a"}
		t2 := emit.Target{Dir: "b", Filename: "foo.go", Package: "b"}
		addEmitPackage(t, ctx, emitPackage("a", emitStructWithFields(
			"a", "A", t1, fieldSpec{name: "F", builtin: "int"},
		)))
		addEmitPackage(t, ctx, emitPackage("b", emitStructWithFields(
			"b", "B", t2, fieldSpec{name: "G", builtin: "string"},
		)))
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if d.HasErrors() {
			t.Fatalf("unexpected diagnostics: %+v", d.Diagnostics())
		}
		if mem.Len() != 2 {
			t.Fatalf("expected 2 sink writes; got %d", mem.Len())
		}
	})
}
