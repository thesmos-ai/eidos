// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package backendtest_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/backendtest"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sink"
)

// TestRun_PassesEmitGraphToBackend pins the happy path: the
// harness seeds the store with the caller's emit packages,
// invokes Backend.Render, and the backend's writes land on the
// returned Sink. The stub backend writes a per-target marker so
// the assertion can confirm the wiring without depending on a
// real template surface.
func TestRun_PassesEmitGraphToBackend(t *testing.T) {
	t.Parallel()

	t.Run("backend renders against pre-built emit packages with Targets resolved", func(t *testing.T) {
		t.Parallel()
		target := emit.Target{Dir: "out", Filename: "x.go", Package: "out"}
		stub := &writingBackend{lang: "stub"}
		result := backendtest.Run(t, backendtest.RunOptions{
			Backend: stub,
			EmitPackages: []*emit.Package{{
				Name: "out", Path: "out",
				Structs: []*emit.Struct{{
					Name: "X", Package: "out", Target: target,
				}},
			}},
		})
		if result.Diag.HasErrors() {
			t.Fatalf("unexpected error diagnostics: %+v", result.Diag.Diagnostics())
		}
		mem, ok := result.Sink.(*sink.Memory)
		if !ok {
			t.Fatalf("expected default *sink.Memory; got %T", result.Sink)
		}
		body, ok := mem.Get(target)
		if !ok {
			t.Fatalf("sink missing the routed target; got files=%v", mem.Files())
		}
		if string(body) != "rendered:out/x.go" {
			t.Fatalf("sink body = %q, want %q", body, "rendered:out/x.go")
		}
	})

	t.Run("supplies the configured language to the backend", func(t *testing.T) {
		t.Parallel()
		stub := &writingBackend{lang: "fixturelang"}
		_ = backendtest.Run(t, backendtest.RunOptions{
			Backend: stub,
			EmitPackages: []*emit.Package{{
				Name: "x", Path: "x",
				Structs: []*emit.Struct{{
					Name: "X", Package: "x",
					Target: emit.Target{Dir: "x", Filename: "x.go", Package: "x"},
				}},
			}},
		})
		if stub.observedLang != "fixturelang" {
			t.Fatalf("backend saw Lang=%q on BackendContext; want %q",
				stub.observedLang, "fixturelang")
		}
	})
}

// writingBackend is the fixture backend the tests use. It writes
// a per-target marker so assertions can verify the harness's
// wiring without depending on a real template surface.
type writingBackend struct {
	lang         string
	observedLang string
}

// Name returns the fixture identifier.
func (*writingBackend) Name() string { return "stub-backend" }

// Language returns the configured language.
func (b *writingBackend) Language() string { return b.lang }

// EmitVersions satisfies the emit-versioned contract every
// backend implements.
func (*writingBackend) EmitVersions() []string { return []string{emit.Major()} }

// Render walks the byTarget index, records the BackendContext's
// configured Lang, and writes a per-target marker through the
// supplied sink.
func (b *writingBackend) Render(ctx *plugin.BackendContext) error {
	b.observedLang = ctx.Lang
	for _, target := range ctx.Store.Emit().ByTarget().Keys() {
		if err := ctx.Sink.Write(target, []byte("rendered:"+target.Dir+"/"+target.Filename)); err != nil {
			return err
		}
	}
	return nil
}
