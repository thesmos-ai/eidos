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

	t.Run("dot-joined ExternalRef.Name maps to underscore-joined Go identifier", func(t *testing.T) {
		t.Parallel()
		// Cross-language frontends (the protobuf frontend, future
		// other-language frontends) surface nested types under the
		// source language's separator. Go identifiers cannot
		// contain dots, so the render-site normalises the dot-
		// joined form to the underscore-joined form matching the
		// protoc-gen-go convention.
		body := renderSingleFieldStruct(t, "P", emit.External("github.com/example/users", "User.Profile"))
		if strings.Contains(body, "User.Profile") {
			t.Fatalf("body should not contain the dot-joined form; got:\n%s", body)
		}
		if !strings.Contains(body, "P users.User_Profile") {
			t.Fatalf("body should contain 'P users.User_Profile'; got:\n%s", body)
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

// TestRenderType_CompositeShapes pins the rendered spelling of
// every documented [emit.CompositeRef.Shape]. Each shape is
// exercised through a struct field's Type so the assertion runs
// through the full template + gofmt pipeline.
func TestRenderType_CompositeShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ref  emit.Ref
		want string
	}{
		{"pointer to int", emit.Ptr(emit.Builtin("int")), "F *int"},
		{"slice of byte", emit.SliceOf(emit.Builtin("byte")), "F []byte"},
		{"array of 16 byte", emit.ArrayOf(emit.Builtin("byte"), 16), "F [16]byte"},
		{"map of string to int", emit.MapOf(emit.Builtin("string"), emit.Builtin("int")), "F map[string]int"},
		{
			name: "func taking int and string returning error",
			ref: emit.FuncOf(
				[]emit.Ref{emit.Builtin("int"), emit.Builtin("string")},
				[]emit.Ref{emit.Builtin("error")},
			),
			want: "F func(int, string) error",
		},
		{
			name: "func with no params or returns",
			ref:  emit.FuncOf(nil, nil),
			want: "F func()",
		},
		{
			name: "func with multi-return",
			ref:  emit.FuncOf([]emit.Ref{emit.Builtin("int")}, []emit.Ref{emit.Builtin("int"), emit.Builtin("error")}),
			want: "F func(int) (int, error)",
		},
		{
			name: "nested pointer to slice",
			ref:  emit.Ptr(emit.SliceOf(emit.Builtin("int"))),
			want: "F *[]int",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := renderSingleFieldStruct(t, "F", tc.ref)
			if !strings.Contains(body, tc.want) {
				t.Fatalf("expected field line to contain %q; got:\n%s", tc.want, body)
			}
		})
	}
}

// TestRenderType_Union covers the ShapeUnion path: type-set
// constraints (`A | B | ~C`) render via the union shape with `~`
// prefixing approximation terms.
func TestRenderType_Union(t *testing.T) {
	t.Parallel()

	t.Run("generic struct with union constraint", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Structs: []*emit.Struct{{
				Name: "Numeric", Package: "x", Target: target,
				TypeParams: []*emit.TypeParam{{
					Name: "T",
					Constraint: &emit.Constraint{
						Embedded: []emit.Ref{
							emit.Union(
								emit.UnionTerm{Type: emit.Builtin("int"), Approx: true},
								emit.UnionTerm{Type: emit.Builtin("float64"), Approx: true},
								emit.UnionTerm{Type: emit.Builtin("string")},
							),
						},
					},
				}},
				Fields: []*emit.Field{{Name: "V", Type: emit.Builtin("int")}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type Numeric[T ~int | ~float64 | string]") {
			t.Fatalf("union constraint mismatched; got:\n%s", body)
		}
	})
}

// TestRenderType_UnknownShape covers the default branch — every
// documented CompositeShape is now wired, so only an out-of-range
// shape value reaches this path.
func TestRenderType_UnknownShape(t *testing.T) {
	t.Parallel()

	t.Run("out-of-range CompositeShape produces an Error diagnostic", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, emitPackage("x", &emit.Struct{
			Name: "X", Package: "x", Target: target,
			Fields: []*emit.Field{{Name: "P", Type: &emit.CompositeRef{Shape: emit.CompositeShape(9999)}}},
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
		if golang.ErrUnsupportedRef == nil {
			t.Fatalf("ErrUnsupportedRef must be exported and non-nil")
		}
		if !errors.Is(golang.ErrUnsupportedRef, golang.ErrUnsupportedRef) {
			t.Fatalf("ErrUnsupportedRef must satisfy errors.Is reflexivity")
		}
	})
}

// TestRenderType_Internal_CrossPackage pins the framework's
// `emit.Internal` cross-package qualification: when the referenced
// target has a resolved [emit.Target.ImportPath] that differs from
// the rendering file's import path, the renderer registers an
// import for the target's package and qualifies the rendered name
// with the resulting alias. This is what lets a generator emit
// `emit.Internal(<mock-struct>)` from a file that routes into a
// different package than the mock (e.g. mocktest's `_test.go`
// file landing in the external test package).
func TestRenderType_Internal_CrossPackage(t *testing.T) {
	t.Parallel()

	t.Run("differing ImportPath qualifies + registers import", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)

		// Target struct lives in package "store" at import path
		// "example.com/x/store". Holder file lives in the external
		// test package at "example.com/x/store_test" — exactly the
		// shape the framework produces after the `_test.go` shift.
		targetStruct := &emit.Struct{
			Name: "SearcherMock", Package: "example.com/x/store",
			Target: emit.Target{
				Dir: "x/store", Filename: "store_mock.go",
				Package: "store", ImportPath: "example.com/x/store",
			},
		}
		holderTarget := emit.Target{
			Dir: "x/store", Filename: "store_mock_test.go",
			Package: "store_test", ImportPath: "example.com/x/store_test",
		}
		holder := &emit.Struct{
			Name: "Holder", Package: "example.com/x/store_test",
			Target: holderTarget,
			Fields: []*emit.Field{{Name: "Inner", Type: emit.Internal(targetStruct)}},
		}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "store", Path: "example.com/x/store",
			Structs: []*emit.Struct{targetStruct},
		})
		addEmitPackage(t, ctx, &emit.Package{
			Name: "store_test", Path: "example.com/x/store_test",
			Structs: []*emit.Struct{holder},
		})
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if d.HasErrors() {
			t.Fatalf("unexpected error diagnostics: %+v", d.Diagnostics())
		}
		body, ok := mem.Get(holderTarget)
		if !ok {
			t.Fatalf("no output for holder target %v", holderTarget)
		}
		s := string(body)
		if !strings.Contains(s, `"example.com/x/store"`) {
			t.Fatalf("body should carry the target's import path; got:\n%s", s)
		}
		if !strings.Contains(s, "Inner store.SearcherMock") {
			t.Fatalf("body should qualify the cross-package ref; got:\n%s", s)
		}
	})

	t.Run("matching ImportPath elides (same-package)", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		shared := emit.Target{
			Dir: "x/store", Filename: "store_main.go",
			Package: "store", ImportPath: "example.com/x/store",
		}
		targetStruct := &emit.Struct{
			Name: "Inner", Package: "example.com/x/store",
			Target: shared,
		}
		holderTarget := emit.Target{
			Dir: "x/store", Filename: "holder.go",
			Package: "store", ImportPath: "example.com/x/store",
		}
		holder := &emit.Struct{
			Name: "Holder", Package: "example.com/x/store",
			Target: holderTarget,
			Fields: []*emit.Field{{Name: "I", Type: emit.Internal(targetStruct)}},
		}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "store", Path: "example.com/x/store",
			Structs: []*emit.Struct{targetStruct, holder},
		})
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if d.HasErrors() {
			t.Fatalf("unexpected error diagnostics: %+v", d.Diagnostics())
		}
		body, ok := mem.Get(holderTarget)
		if !ok {
			t.Fatalf("no output for holder")
		}
		s := string(body)
		if strings.Contains(s, "store.Inner") {
			t.Fatalf("same-package ref should elide qualifier; got:\n%s", s)
		}
		if !strings.Contains(s, "I Inner") {
			t.Fatalf("body should carry bare 'Inner'; got:\n%s", s)
		}
	})
}
