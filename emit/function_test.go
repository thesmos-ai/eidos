// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func makeFunction() *emit.Function {
	return &emit.Function{
		Name:    "Open",
		Package: "users",
		Params: []*emit.Param{
			{Name: "path", Type: builtinRef("string")},
		},
		Returns: []emit.Ref{builtinRef("error")},
	}
}

func TestFunction_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindFunction", func(t *testing.T) {
		t.Parallel()
		var f emit.Function
		if f.Kind() != emit.KindFunction {
			t.Fatalf("Kind = %s, want %s", f.Kind(), emit.KindFunction)
		}
	})
}

func TestFunction_QName(t *testing.T) {
	t.Parallel()

	t.Run("includes package when present", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, makeFunction().QName(), "users.Open")
	})

	t.Run("returns just the name when package is empty", func(t *testing.T) {
		t.Parallel()
		assertEqualString(t, (&emit.Function{Name: "Open"}).QName(), "Open")
	})
}

func TestFunction_IsVariadic(t *testing.T) {
	t.Parallel()

	t.Run("returns true when last param is variadic", func(t *testing.T) {
		t.Parallel()
		f := &emit.Function{
			Params: []*emit.Param{
				{Name: "args", Type: builtinRef("string"), Variadic: true},
			},
		}
		if !f.IsVariadic() {
			t.Fatalf("variadic function should report IsVariadic true")
		}
	})

	t.Run("returns false otherwise", func(t *testing.T) {
		t.Parallel()
		if makeFunction().IsVariadic() {
			t.Fatalf("non-variadic function should report IsVariadic false")
		}
	})

	t.Run("returns false on empty params", func(t *testing.T) {
		t.Parallel()
		if (&emit.Function{}).IsVariadic() {
			t.Fatalf("empty-params function should report IsVariadic false")
		}
	})
}

func TestFunction_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("reports true when type params declared", func(t *testing.T) {
		t.Parallel()
		f := &emit.Function{TypeParams: []*emit.TypeParam{{Name: "T"}}}
		if !f.IsGeneric() {
			t.Fatalf("generic function should report IsGeneric true")
		}
	})

	t.Run("reports false otherwise", func(t *testing.T) {
		t.Parallel()
		if (&emit.Function{}).IsGeneric() {
			t.Fatalf("non-generic function should report IsGeneric false")
		}
	})
}

func TestFunction_ParamCount(t *testing.T) {
	t.Parallel()

	t.Run("returns the parameter count", func(t *testing.T) {
		t.Parallel()
		if got := makeFunction().ParamCount(); got != 1 {
			t.Fatalf("ParamCount = %d, want 1", got)
		}
	})
}

func TestFunction_ReturnCount(t *testing.T) {
	t.Parallel()

	t.Run("returns the return count", func(t *testing.T) {
		t.Parallel()
		if got := makeFunction().ReturnCount(); got != 1 {
			t.Fatalf("ReturnCount = %d, want 1", got)
		}
	})
}

func TestFunction_ParamByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching parameter", func(t *testing.T) {
		t.Parallel()
		got := makeFunction().ParamByName("path")
		if got == nil || got.Name != "path" {
			t.Fatalf("ParamByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makeFunction().ParamByName("missing") != nil {
			t.Fatalf("ParamByName(unknown) should be nil")
		}
	})

	t.Run("returns nil for empty name", func(t *testing.T) {
		t.Parallel()
		f := &emit.Function{Params: []*emit.Param{{Type: builtinRef("int")}}}
		if f.ParamByName("") != nil {
			t.Fatalf("ParamByName(\"\") should not match anonymous params")
		}
	})
}

func TestFunction_ParamAt(t *testing.T) {
	t.Parallel()

	t.Run("returns the parameter at the given index", func(t *testing.T) {
		t.Parallel()
		if got := makeFunction().ParamAt(0); got == nil || got.Name != "path" {
			t.Fatalf("ParamAt(0) mismatch: %+v", got)
		}
	})

	t.Run("returns nil for out-of-range indexes", func(t *testing.T) {
		t.Parallel()
		f := makeFunction()
		if f.ParamAt(-1) != nil || f.ParamAt(99) != nil {
			t.Fatalf("ParamAt out-of-range should return nil")
		}
	})
}

func TestFunction_ReturnAt(t *testing.T) {
	t.Parallel()

	t.Run("returns the return type at the given index", func(t *testing.T) {
		t.Parallel()
		if got := makeFunction().ReturnAt(0); got == nil {
			t.Fatalf("ReturnAt(0) should be non-nil")
		}
	})

	t.Run("returns nil for out-of-range indexes", func(t *testing.T) {
		t.Parallel()
		f := makeFunction()
		if f.ReturnAt(-1) != nil || f.ReturnAt(99) != nil {
			t.Fatalf("ReturnAt out-of-range should return nil")
		}
	})
}

func TestFunction_Slots(t *testing.T) {
	t.Parallel()

	t.Run("Prebody / Postbody / ParamsSlot / ReturnsSlot / Slot are distinct and idempotent", func(t *testing.T) {
		t.Parallel()
		f := makeFunction()
		pre1, pre2 := f.Prebody(), f.Prebody()
		post1, post2 := f.Postbody(), f.Postbody()
		ps1, ps2 := f.ParamsSlot(), f.ParamsSlot()
		rs1, rs2 := f.ReturnsSlot(), f.ReturnsSlot()
		c1, c2 := f.Slot("custom"), f.Slot("custom")
		if pre1 != pre2 || post1 != post2 || ps1 != ps2 || rs1 != rs2 || c1 != c2 {
			t.Fatalf("slot lookups should be idempotent")
		}
		if pre1 == post1 {
			t.Fatalf("prebody and postbody must differ")
		}
	})
}
