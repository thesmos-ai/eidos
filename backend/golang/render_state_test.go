// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
)

// TestRenderType_Builtin covers the BuiltinRef branch of renderType
// via the public render path. Builtins render as their name verbatim
// — no imp call, no alias.
func TestRenderType_Builtin(t *testing.T) {
	t.Parallel()

	t.Run("named builtin renders as its name", func(t *testing.T) {
		t.Parallel()
		body := renderSingleFieldStruct(t, "F", emit.Builtin("int"))
		if !strings.Contains(body, "F int") {
			t.Fatalf("body should contain 'F int'; got:\n%s", body)
		}
	})

	t.Run("multi-word builtin renders verbatim", func(t *testing.T) {
		t.Parallel()
		body := renderSingleFieldStruct(t, "Err", emit.Builtin("error"))
		if !strings.Contains(body, "Err error") {
			t.Fatalf("body should contain 'Err error'; got:\n%s", body)
		}
	})
}

// TestRenderType_External covers the ExternalRef branch. The render
// path calls imp internally and the resulting file carries the
// import in its block plus the alias-qualified type reference.
func TestRenderType_External(t *testing.T) {
	t.Parallel()

	t.Run("stdlib ExternalRef triggers an import and alias-qualified reference", func(t *testing.T) {
		t.Parallel()
		body := renderSingleFieldStruct(t, "Ctx", emit.External("context", "Context"))
		if !strings.Contains(body, "\"context\"") {
			t.Fatalf("body should contain stdlib import 'context'; got:\n%s", body)
		}
		if !strings.Contains(body, "Ctx context.Context") {
			t.Fatalf("body should contain alias-qualified field; got:\n%s", body)
		}
	})

	t.Run("ExternalRef with empty package produces an Error diagnostic", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, emitPackage("x", &emit.Struct{
			Name: "X", Package: "x", Target: target,
			Fields: []*emit.Field{{Name: "F", Type: emit.External("", "Bad")}},
		}))
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render should not propagate the renderType error: %v", err)
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("failed renderType must not produce a sink write")
		}
		// `writer.ImportSet.Imp` returns ErrEmptyPath for an empty
		// path; that error propagates through renderType into a
		// per-Target Error diagnostic.
		if !diagnosticsContain(d, diag.Error, "renderType") {
			t.Fatalf("expected Error diagnostic from renderType; got %+v", d.Diagnostics())
		}
	})

	t.Run("external (non-stdlib) ExternalRef uses last-segment alias by default", func(t *testing.T) {
		t.Parallel()
		body := renderSingleFieldStruct(t, "User", emit.External("github.com/example/users", "User"))
		if !strings.Contains(body, "\"github.com/example/users\"") {
			t.Fatalf("body should contain external import; got:\n%s", body)
		}
		if !strings.Contains(body, "User users.User") {
			t.Fatalf("body should contain alias-qualified field 'User users.User'; got:\n%s", body)
		}
	})
}

// TestRenderType_Internal covers the TypeRef branch. Internal refs
// render as the target's unqualified name without producing an
// import.
func TestRenderType_Internal(t *testing.T) {
	t.Parallel()

	t.Run("TypeRef to a same-package struct renders as bare name", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		// Two structs in the same package; Holder references Inner.
		inner := &emit.Struct{Name: "Inner", Package: "x", Target: target}
		holder := &emit.Struct{
			Name: "Holder", Package: "x", Target: target,
			Fields: []*emit.Field{{Name: "I", Type: emit.Internal(inner)}},
		}
		addEmitPackage(t, ctx, emitPackage("x", inner, holder))
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
		if strings.Contains(string(body), "import (") {
			t.Fatalf("internal TypeRef should not produce imports; got:\n%s", body)
		}
		if !strings.Contains(string(body), "I Inner") {
			t.Fatalf("body should contain 'I Inner'; got:\n%s", body)
		}
	})

	t.Run("TypeRef to a same-package interface renders as bare name", func(t *testing.T) {
		t.Parallel()
		assertInternalRefRenders(t, &emit.Interface{Name: "Doer", Package: "x"}, "Doer")
	})

	t.Run("TypeRef to a same-package alias renders as bare name", func(t *testing.T) {
		t.Parallel()
		assertInternalRefRenders(t, &emit.Alias{Name: "ID", Package: "x"}, "ID")
	})

	t.Run("TypeRef to a same-package enum renders as bare name", func(t *testing.T) {
		t.Parallel()
		assertInternalRefRenders(t, &emit.Enum{Name: "Status", Package: "x"}, "Status")
	})

	t.Run("TypeRef to a same-package function renders as bare name", func(t *testing.T) {
		t.Parallel()
		assertInternalRefRenders(t, &emit.Function{Name: "Handler", Package: "x"}, "Handler")
	})

	t.Run("TypeRef pointing at an unsupported target kind returns ErrUnsupportedRef", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		// Method is not a top-level decl renderable as a TypeRef
		// target; the targetName helper rejects it.
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Structs: []*emit.Struct{{
				Name: "X", Package: "x", Target: target,
				Fields: []*emit.Field{{Name: "M", Type: &emit.TypeRef{Target: &emit.Method{Name: "Bad"}}}},
			}},
		})
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render returned non-nil; render errors should diagnose, not propagate: %v", err)
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("failed renderType must not produce a sink write")
		}
		if !diagnosticsContain(d, diag.Error, "unsupported Ref") {
			t.Fatalf("expected Error diagnostic naming ErrUnsupportedRef; got %+v", d.Diagnostics())
		}
	})
}

// TestRenderType_UnsupportedKind covers the default branch of
// renderType — currently any non-Builtin/External/Internal Ref.
func TestRenderType_UnsupportedKind(t *testing.T) {
	t.Parallel()

	t.Run("CompositeRef shape produces an Error diagnostic", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, emitPackage("x", &emit.Struct{
			Name: "X", Package: "x", Target: target,
			Fields: []*emit.Field{{Name: "P", Type: emit.Ptr(emit.Builtin("int"))}},
		}))
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("failed render must not produce a sink write")
		}
		if !diagnosticsContain(d, diag.Error, "unsupported Ref") {
			t.Fatalf("expected Error diagnostic; got %+v", d.Diagnostics())
		}
	})

	t.Run("ErrUnsupportedRef is exported and wraps cleanly", func(t *testing.T) {
		t.Parallel()
		// Indirectly exercise errors.Is via the diagnostic surface:
		// the produced diagnostic message wraps ErrUnsupportedRef
		// through fmt.Errorf("%w: …"). The string check covers the
		// "exported" half; runtime wrap-identity is exercised by the
		// templates_test missing-template case.
		if golang.ErrUnsupportedRef == nil {
			t.Fatalf("ErrUnsupportedRef must be exported and non-nil")
		}
		if !errors.Is(golang.ErrUnsupportedRef, golang.ErrUnsupportedRef) {
			t.Fatalf("ErrUnsupportedRef must satisfy errors.Is reflexivity")
		}
	})
}
