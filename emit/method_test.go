// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func makeMethod() *emit.Method {
	return &emit.Method{
		Name:         "Save",
		Receiver:     emit.Ptr(emit.Builtin("Repo")),
		ReceiverName: "r",
		Params: []*emit.Param{
			{Name: "ctx", Type: externalRef("context", "Context")},
			{Name: "user", Type: emit.Ptr(emit.Builtin("User"))},
		},
		Returns: []emit.Ref{builtinRef("error")},
	}
}

func TestMethod_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindMethod", func(t *testing.T) {
		t.Parallel()
		var m emit.Method
		if m.Kind() != emit.KindMethod {
			t.Fatalf("Kind = %s, want %s", m.Kind(), emit.KindMethod)
		}
	})
}

func TestMethod_HasReceiver(t *testing.T) {
	t.Parallel()

	t.Run("reports true when Receiver is non-nil", func(t *testing.T) {
		t.Parallel()
		if !makeMethod().HasReceiver() {
			t.Fatalf("struct method should have a receiver")
		}
	})

	t.Run("reports false when Receiver is nil", func(t *testing.T) {
		t.Parallel()
		m := &emit.Method{Name: "M"}
		if m.HasReceiver() {
			t.Fatalf("interface method should not have a receiver")
		}
	})
}

func TestMethod_IsVariadic(t *testing.T) {
	t.Parallel()

	t.Run("returns true when last param is variadic", func(t *testing.T) {
		t.Parallel()
		m := &emit.Method{
			Params: []*emit.Param{
				{Name: "first", Type: builtinRef("int")},
				{Name: "rest", Type: builtinRef("int"), Variadic: true},
			},
		}
		if !m.IsVariadic() {
			t.Fatalf("method with variadic last param should report IsVariadic")
		}
	})

	t.Run("returns false when last param is not variadic", func(t *testing.T) {
		t.Parallel()
		if makeMethod().IsVariadic() {
			t.Fatalf("non-variadic method should report IsVariadic false")
		}
	})

	t.Run("returns false when params list is empty", func(t *testing.T) {
		t.Parallel()
		m := &emit.Method{}
		if m.IsVariadic() {
			t.Fatalf("empty-params method should report IsVariadic false")
		}
	})
}

func TestMethod_IsGeneric(t *testing.T) {
	t.Parallel()

	t.Run("reports true when type params declared", func(t *testing.T) {
		t.Parallel()
		m := &emit.Method{TypeParams: []*emit.TypeParam{{Name: "T"}}}
		if !m.IsGeneric() {
			t.Fatalf("generic method should report IsGeneric true")
		}
	})

	t.Run("reports false otherwise", func(t *testing.T) {
		t.Parallel()
		if (&emit.Method{}).IsGeneric() {
			t.Fatalf("non-generic method should report IsGeneric false")
		}
	})
}

func TestMethod_ParamCount(t *testing.T) {
	t.Parallel()

	t.Run("returns the number of declared parameters", func(t *testing.T) {
		t.Parallel()
		if got := makeMethod().ParamCount(); got != 2 {
			t.Fatalf("ParamCount = %d, want 2", got)
		}
	})
}

func TestMethod_ReturnCount(t *testing.T) {
	t.Parallel()

	t.Run("returns the number of declared returns", func(t *testing.T) {
		t.Parallel()
		if got := makeMethod().ReturnCount(); got != 1 {
			t.Fatalf("ReturnCount = %d, want 1", got)
		}
	})
}

func TestMethod_ParamByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching parameter", func(t *testing.T) {
		t.Parallel()
		got := makeMethod().ParamByName("ctx")
		if got == nil || got.Name != "ctx" {
			t.Fatalf("ParamByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makeMethod().ParamByName("missing") != nil {
			t.Fatalf("ParamByName(unknown) should be nil")
		}
	})

	t.Run("returns nil for an empty name to avoid matching anonymous params", func(t *testing.T) {
		t.Parallel()
		m := &emit.Method{Params: []*emit.Param{{Type: builtinRef("int")}}}
		if m.ParamByName("") != nil {
			t.Fatalf("ParamByName(\"\") should not match anonymous params")
		}
	})
}

func TestMethod_ParamAt(t *testing.T) {
	t.Parallel()

	t.Run("returns the parameter at the given index", func(t *testing.T) {
		t.Parallel()
		if got := makeMethod().ParamAt(1); got == nil || got.Name != "user" {
			t.Fatalf("ParamAt(1) mismatch: %+v", got)
		}
	})

	t.Run("returns nil for out-of-range indexes", func(t *testing.T) {
		t.Parallel()
		m := makeMethod()
		if m.ParamAt(-1) != nil || m.ParamAt(99) != nil {
			t.Fatalf("ParamAt out-of-range should return nil")
		}
	})
}

func TestMethod_ReturnAt(t *testing.T) {
	t.Parallel()

	t.Run("returns the return type at the given index", func(t *testing.T) {
		t.Parallel()
		if got := makeMethod().ReturnAt(0); got == nil {
			t.Fatalf("ReturnAt(0) should return the first return type")
		}
	})

	t.Run("returns nil for out-of-range indexes", func(t *testing.T) {
		t.Parallel()
		m := makeMethod()
		if m.ReturnAt(-1) != nil || m.ReturnAt(99) != nil {
			t.Fatalf("ReturnAt out-of-range should return nil")
		}
	})
}

func TestMethod_Slots(t *testing.T) {
	t.Parallel()

	t.Run("Prebody and Postbody slots are independent and accept Stmts", func(t *testing.T) {
		t.Parallel()
		m := makeMethod()
		pre := m.Prebody()
		post := m.Postbody()
		if pre == post {
			t.Fatalf("Prebody and Postbody must be distinct slots")
		}
		assertNoError(t, pre.Append(emit.NewExprStmt(emit.NewIdent("x")), emit.Provenance{}))
		assertNoError(t, post.Append(emit.NewExprStmt(emit.NewIdent("y")), emit.Provenance{}))
	})

	t.Run("ParamsSlot, ReturnsSlot and Slot lookups are idempotent", func(t *testing.T) {
		t.Parallel()
		m := makeMethod()
		if a, b := m.ParamsSlot(), m.ParamsSlot(); a != b {
			t.Fatalf("ParamsSlot should be idempotent")
		}
		if a, b := m.ReturnsSlot(), m.ReturnsSlot(); a != b {
			t.Fatalf("ReturnsSlot should be idempotent")
		}
		if a, b := m.Slot("x"), m.Slot("x"); a != b {
			t.Fatalf("Slot lookup should be idempotent")
		}
	})
}
