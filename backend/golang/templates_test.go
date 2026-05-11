// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/emit"
)

// TestTemplates_DispatchByKind verifies the `render` funcmap entry
// dispatches by [emit.Node.Kind], producing the matching kind's
// rendered output inline within a parent template.
func TestTemplates_DispatchByKind(t *testing.T) {
	t.Parallel()

	t.Run("emit.struct rendered through emit.file's render call", func(t *testing.T) {
		t.Parallel()
		body := renderSingleFieldStruct(t, "Name", emit.Builtin("string"))
		// The file template's `package <pkg>` clause should be
		// followed by the struct template's `type X struct { ... }`
		// block. Verifying both fragments are present is the
		// dispatch-level assertion.
		if !strings.Contains(body, "package x") {
			t.Fatalf("body should contain 'package x'; got:\n%s", body)
		}
		if !strings.Contains(body, "type X struct {") {
			t.Fatalf("body should contain 'type X struct {'; got:\n%s", body)
		}
	})
}

// TestTemplates_MissingKindSurfacesAsDiagnostic covers the
// no-template-for-kind case. The contract: render-dispatch
// produces an Error diagnostic naming the kind and the loop
// continues with the next Target.
func TestTemplates_MissingKindSurfacesAsDiagnostic(t *testing.T) {
	t.Parallel()

	t.Run("unsupported kind in a slot would surface ErrTemplateMissing", func(t *testing.T) {
		t.Parallel()
		// We can't yet exercise a missing-kind dispatch through the
		// public surface without slot rendering or plugin-defined
		// kinds. The earliest reachable miss-path is render via the
		// emit.file template attempting to dispatch into a kind
		// without a registered template. The current core kinds
		// supported are emit.struct and emit.file; routing an
		// emit.Function through emit.file dispatches to a missing
		// "emit.function" template (it lands in Phase E2).
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "out", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Functions: []*emit.Function{{Name: "F", Package: "x", Target: target}},
		})
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("missing-kind render must not produce a sink write")
		}
		if !diagnosticsContain(d, diag.Error, "emit.function") {
			t.Fatalf("expected Error diagnostic naming the missing kind; got %+v", d.Diagnostics())
		}
	})
}
