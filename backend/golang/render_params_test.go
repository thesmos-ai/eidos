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

// TestRenderParams_Variadic covers the variadic-parameter branch of
// renderParams — `...` prefix on the last parameter's type.
func TestRenderParams_Variadic(t *testing.T) {
	t.Parallel()
	t.Run("variadic param renders with '...' prefix", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Interfaces: []*emit.Interface{{
				Name: "Printer", Package: "x", Target: target,
				Methods: []*emit.Method{
					{Name: "Print", Params: []*emit.Param{
						{Name: "args", Type: emit.Builtin("string"), Variadic: true},
					}},
				},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "Print(args ...string)") {
			t.Fatalf("variadic param mismatched; got:\n%s", body)
		}
	})
}

// TestRenderParams_MixedNamed covers the [ErrMixedNamedParams]
// validation: a parameter list mixing named and unnamed entries
// violates Go's grammar and surfaces as an Error diagnostic.
func TestRenderParams_MixedNamed(t *testing.T) {
	t.Parallel()

	t.Run("mixed-named param list produces ErrMixedNamedParams diagnostic", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Interfaces: []*emit.Interface{{
				Name: "I", Package: "x", Target: target,
				Methods: []*emit.Method{
					{Name: "M", Params: []*emit.Param{
						{Name: "a", Type: emit.Builtin("int")},
						{Type: emit.Builtin("string")},
					}},
				},
			}},
		})
		if err := mustNew(t).Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if _, ok := mem.Get(target); ok {
			t.Fatalf("render must not produce output on mixed-named params")
		}
		if !diagnosticsContain(d, diag.Error, "mixes named and unnamed") {
			t.Fatalf("expected ErrMixedNamedParams diagnostic; got %+v", d.Diagnostics())
		}
	})

	t.Run("ErrMixedNamedParams is exported", func(t *testing.T) {
		t.Parallel()
		if golang.ErrMixedNamedParams == nil {
			t.Fatalf("ErrMixedNamedParams must be exported and non-nil")
		}
		if !errors.Is(golang.ErrMixedNamedParams, golang.ErrMixedNamedParams) {
			t.Fatalf("ErrMixedNamedParams must satisfy errors.Is reflexivity")
		}
	})
}
