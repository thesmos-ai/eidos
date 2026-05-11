// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"bytes"
	"go/format"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
)

// TestFormat_HappyPath covers the format step's contract: rendered
// output passes go/format.Source cleanly and the sink receives the
// formatted bytes.
func TestFormat_HappyPath(t *testing.T) {
	t.Parallel()

	t.Run("rendered struct is gofmt-clean", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "out", Filename: "user.go", Package: "users"}
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
		// Round-trip through format.Source: the body is fixed-point
		// under gofmt, which is the testable equivalent of "passes
		// go/format.Source cleanly".
		formatted, err := format.Source(body)
		if err != nil {
			t.Fatalf("format.Source rejected backend output: %v\noutput:\n%s", err, body)
		}
		if !bytes.Equal(formatted, body) {
			t.Fatalf("backend output is not gofmt-stable\n--- got ---\n%s\n--- gofmt ---\n%s", body, formatted)
		}
	})

	t.Run("repeated runs of the same fixture produce byte-identical output", func(t *testing.T) {
		t.Parallel()
		first := renderSingleFieldStruct(t, "ID", emit.Builtin("int"))
		second := renderSingleFieldStruct(t, "ID", emit.Builtin("int"))
		if first != second {
			t.Fatalf("output is not byte-identical across runs\n--- first ---\n%s\n--- second ---\n%s", first, second)
		}
	})
}

// TestFormat_FailurePath covers the warn-on-failure contract:
// inducing a format failure produces a Warn diagnostic on the
// target and writes the unformatted body through the sink so the
// user can debug.
func TestFormat_FailurePath(t *testing.T) {
	t.Parallel()

	t.Run("malformed template output surfaces a Warn and writes unformatted bytes", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "out", Filename: "x.go", Package: "x"}
		// Field names containing whitespace are illegal Go
		// identifiers — the template renders them verbatim and
		// format.Source rejects the result. The backend's contract
		// is to warn-and-emit-unformatted, not to abort.
		addEmitPackage(t, ctx, emitPackage("x", &emit.Struct{
			Name:    "X",
			Package: "x",
			Target:  target,
			Fields:  []*emit.Field{{Name: "Not A Valid Name", Type: emit.Builtin("int")}},
		}))
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render should not return an error for a format failure: %v", err)
		}
		body, ok := mem.Get(target)
		if !ok {
			t.Fatalf("warn-on-failure contract: sink must still receive the unformatted body")
		}
		if !strings.Contains(string(body), "Not A Valid Name") {
			t.Fatalf("body should retain the unformatted content; got:\n%s", body)
		}
		if !diagnosticsContain(d, diag.Warn, "format.Source failed") {
			t.Fatalf("expected Warn diagnostic from format.Source failure; got %+v", d.Diagnostics())
		}
	})
}
