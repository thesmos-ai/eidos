// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture_test

import (
	"testing"

	"go.thesmos.sh/eidos/storefixture"
	"go.thesmos.sh/eidos/node"
)

// requireQName fails the test if the produced node's qualified name
// does not match want — a frequent assertion across every kind of
// declaration the fixture produces.
func requireQName(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("qname mismatch: got %q, want %q", got, want)
	}
}

// requirePanic runs fn and fails the test if it does not panic.
// Returns the recovered value so callers may assert against it.
func requirePanic(t *testing.T, fn func()) (rec any) {
	t.Helper()
	defer func() {
		rec = recover()
		if rec == nil {
			t.Fatalf("expected panic; fn returned normally")
		}
	}()
	fn()
	return nil
}

// asNamedRef asserts that t is a Named [node.TypeRef] and returns it.
// Used by tests that need to inspect a constructed type ref by kind.
func asNamedRef(t *testing.T, r *node.TypeRef) *node.TypeRef {
	t.Helper()
	if r == nil || r.TypeKind != node.TypeRefNamed {
		t.Fatalf("expected Named ref, got %+v", r)
	}
	return r
}

// captureFirstMethod runs fn against a fresh struct's method builder
// and returns the resulting [node.Method]. Used by method-builder
// tests that exercise a single method's configuration in isolation.
func captureFirstMethod(t *testing.T, fn func(*storefixture.MethodBuilder)) *node.Method {
	t.Helper()
	var captured *node.Method
	storefixture.New().Struct("S", func(b *storefixture.StructBuilder) {
		b.Method("M", func(m *storefixture.MethodBuilder) {
			fn(m)
			captured = m.Node()
		})
	})
	return captured
}

// captureFirstFunction runs fn against a fresh function builder and
// returns the resulting [node.Function]. Used by function-builder
// tests that exercise a single function's configuration in isolation.
func captureFirstFunction(t *testing.T, fn func(*storefixture.FunctionBuilder)) *node.Function {
	t.Helper()
	var captured *node.Function
	storefixture.New().Function("F", func(b *storefixture.FunctionBuilder) {
		fn(b)
		captured = b.Node()
	})
	return captured
}

// captureFirstField builds a struct with one field and returns the
// resulting [node.Field] after running fn against the field builder.
func captureFirstField(t *testing.T, fn func(*storefixture.FieldBuilder)) *node.Field {
	t.Helper()
	var captured *node.Field
	storefixture.New().Struct("S", func(b *storefixture.StructBuilder) {
		b.Field("F", storefixture.Named("string"), func(fb *storefixture.FieldBuilder) {
			fn(fb)
			captured = fb.Node()
		})
	})
	return captured
}

// captureFirstVariable runs fn against a fresh variable builder and
// returns the resulting [node.Variable].
func captureFirstVariable(t *testing.T, fn func(*storefixture.VariableBuilder)) *node.Variable {
	t.Helper()
	var captured *node.Variable
	storefixture.New().Variable("V", func(b *storefixture.VariableBuilder) {
		fn(b)
		captured = b.Node()
	})
	return captured
}

// captureFirstConstant runs fn against a fresh constant builder and
// returns the resulting [node.Constant].
func captureFirstConstant(t *testing.T, fn func(*storefixture.ConstantBuilder)) *node.Constant {
	t.Helper()
	var captured *node.Constant
	storefixture.New().Constant("C", func(b *storefixture.ConstantBuilder) {
		fn(b)
		captured = b.Node()
	})
	return captured
}

// captureFirstEnum runs fn against a fresh enum builder and returns
// the resulting [node.Enum].
func captureFirstEnum(t *testing.T, fn func(*storefixture.EnumBuilder)) *node.Enum {
	t.Helper()
	var captured *node.Enum
	storefixture.New().Enum("E", func(b *storefixture.EnumBuilder) {
		fn(b)
		captured = b.Node()
	})
	return captured
}

// captureFirstAlias runs fn against a fresh alias builder and returns
// the resulting [node.Alias].
func captureFirstAlias(t *testing.T, fn func(*storefixture.AliasBuilder)) *node.Alias {
	t.Helper()
	var captured *node.Alias
	storefixture.New().Alias("A", func(b *storefixture.AliasBuilder) {
		fn(b)
		captured = b.Node()
	})
	return captured
}
