// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
)

// TestRenderType covers the type-rendering funcmap entry through
// the backend's public render path. The supported surface widens
// phase by phase; this test pins the current contract.
func TestRenderType(t *testing.T) {
	t.Parallel()

	t.Run("BuiltinRef renders as its name", func(t *testing.T) {
		t.Parallel()
		body := renderSingleFieldStruct(t, "F", emit.Builtin("int"))
		if !strings.Contains(body, "F int") {
			t.Fatalf("rendered struct should contain 'F int'; got:\n%s", body)
		}
	})

	t.Run("BuiltinRef with multi-word name renders verbatim", func(t *testing.T) {
		t.Parallel()
		body := renderSingleFieldStruct(t, "Err", emit.Builtin("error"))
		if !strings.Contains(body, "Err error") {
			t.Fatalf("rendered struct should contain 'Err error'; got:\n%s", body)
		}
	})

	t.Run("unsupported Ref produces an Error diagnostic naming ErrUnsupportedRef", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "out", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, emitPackage("x", &emit.Struct{
			Name:    "X",
			Package: "x",
			Target:  target,
			Fields:  []*emit.Field{{Name: "P", Type: emit.Ptr(emit.Builtin("int"))}},
		}))
		be := mustNew(t)
		if err := be.Render(ctx); err != nil {
			t.Fatalf("Render returned non-nil error; render errors should surface as diagnostics, got: %v", err)
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("failed render must not produce a sink write for target %v", target)
		}
		if !diagnosticsContain(d, diag.Error, "unsupported Ref") {
			t.Fatalf("expected Error diagnostic mentioning unsupported Ref; got %+v", d.Diagnostics())
		}
	})
}

// renderSingleFieldStruct builds a one-field struct whose field
// type is r, runs the full backend render path, and returns the
// rendered file body. Fails the test on any diagnostic or sink
// miss — happy-path-only helper.
func renderSingleFieldStruct(t *testing.T, fieldName string, r emit.Ref) string {
	t.Helper()
	ctx, mem, d := newBackendContext(t)
	target := emit.Target{Dir: "out", Filename: "x.go", Package: "x"}
	addEmitPackage(t, ctx, emitPackage("x", &emit.Struct{
		Name:    "X",
		Package: "x",
		Target:  target,
		Fields:  []*emit.Field{{Name: fieldName, Type: r}},
	}))
	be := mustNew(t)
	if err := be.Render(ctx); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if d.HasErrors() {
		t.Fatalf("unexpected error diagnostics: %+v", d.Diagnostics())
	}
	body, ok := mem.Get(target)
	if !ok {
		t.Fatalf("backend produced no output for target %v", target)
	}
	return string(body)
}

// mustNew constructs a Backend. Trivial wrapper used by tests so
// switching construction patterns (constructor → factory, etc.)
// touches one site rather than every test file.
func mustNew(t *testing.T) *golang.Backend {
	t.Helper()
	return golang.New()
}

// diagnosticsContain reports whether d carries at least one
// diagnostic at the given severity whose message contains substr.
func diagnosticsContain(d *diag.Sink, sev diag.Severity, substr string) bool {
	for _, dg := range d.Diagnostics() {
		if dg.Severity == sev && strings.Contains(dg.Message, substr) {
			return true
		}
	}
	return false
}
