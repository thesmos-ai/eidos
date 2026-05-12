// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
)

// TestTypeArgsFromNodeParams covers the node-layer lifting helper:
// an empty input returns nil, a populated input produces parallel
// bare-name [emit.Builtin] refs preserving order.
func TestTypeArgsFromNodeParams(t *testing.T) {
	t.Parallel()

	t.Run("nil and empty inputs return nil", func(t *testing.T) {
		t.Parallel()
		if got := builder.TypeArgsFromNodeParams(nil); got != nil {
			t.Fatalf("nil input = %v, want nil", got)
		}
		if got := builder.TypeArgsFromNodeParams([]*node.TypeParam{}); got != nil {
			t.Fatalf("empty input = %v, want nil", got)
		}
	})

	t.Run("populated input lifts to parallel Builtin refs", func(t *testing.T) {
		t.Parallel()
		in := []*node.TypeParam{{Name: "T"}, {Name: "K"}, {Name: "V"}}
		out := builder.TypeArgsFromNodeParams(in)
		if len(out) != 3 {
			t.Fatalf("len(out) = %d, want 3", len(out))
		}
		for i, want := range []string{"T", "K", "V"} {
			b, ok := out[i].(*emit.BuiltinRef)
			if !ok {
				t.Fatalf("out[%d] = %T, want *emit.BuiltinRef", i, out[i])
			}
			if b.Name != want {
				t.Fatalf("out[%d].Name = %q, want %q", i, b.Name, want)
			}
		}
	})
}

// TestTypeArgsFromEmitParams mirrors the node-layer test for the
// emit-layer input shape.
func TestTypeArgsFromEmitParams(t *testing.T) {
	t.Parallel()

	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		if got := builder.TypeArgsFromEmitParams(nil); got != nil {
			t.Fatalf("nil input = %v, want nil", got)
		}
	})

	t.Run("populated input lifts to parallel Builtin refs", func(t *testing.T) {
		t.Parallel()
		in := []*emit.TypeParam{{Name: "A"}, {Name: "B"}}
		out := builder.TypeArgsFromEmitParams(in)
		if len(out) != 2 {
			t.Fatalf("len(out) = %d, want 2", len(out))
		}
		for i, want := range []string{"A", "B"} {
			b, ok := out[i].(*emit.BuiltinRef)
			if !ok {
				t.Fatalf("out[%d] = %T, want *emit.BuiltinRef", i, out[i])
			}
			if b.Name != want {
				t.Fatalf("out[%d].Name = %q, want %q", i, b.Name, want)
			}
		}
	})
}

// TestApplyTypeArgs covers the ref-adapter: External refs gain
// the supplied type args; other ref variants pass through; empty
// typeArgs leaves the ref untouched.
func TestApplyTypeArgs(t *testing.T) {
	t.Parallel()

	t.Run("empty typeArgs leaves the ref unchanged", func(t *testing.T) {
		t.Parallel()
		in := emit.External("pkg", "Foo")
		if got := builder.ApplyTypeArgs(in, nil); got != in {
			t.Fatalf("nil typeArgs should pass through; got %v", got)
		}
		if got := builder.ApplyTypeArgs(in, []emit.Ref{}); got != in {
			t.Fatalf("empty typeArgs should pass through; got %v", got)
		}
	})

	t.Run("ExternalRef receives a fresh ref with the type args attached", func(t *testing.T) {
		t.Parallel()
		args := []emit.Ref{emit.Builtin("T")}
		got := builder.ApplyTypeArgs(emit.External("pkg", "Foo"), args)
		ext, ok := got.(*emit.ExternalRef)
		if !ok {
			t.Fatalf("ApplyTypeArgs returned %T, want *emit.ExternalRef", got)
		}
		if ext.Package != "pkg" || ext.Name != "Foo" {
			t.Fatalf("Package/Name lost: %+v", ext)
		}
		if len(ext.TypeArgs) != 1 {
			t.Fatalf("TypeArgs = %v, want one entry", ext.TypeArgs)
		}
	})

	t.Run("non-ExternalRef passes through unchanged", func(t *testing.T) {
		t.Parallel()
		in := emit.Builtin("string")
		got := builder.ApplyTypeArgs(in, []emit.Ref{emit.Builtin("T")})
		if got != in {
			t.Fatalf("Builtin should pass through; got %v", got)
		}
	})
}
