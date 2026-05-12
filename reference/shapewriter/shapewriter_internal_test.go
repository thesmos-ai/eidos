// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shapewriter

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
)

// TestOnStruct_OverrideMatrix covers every override-vs-heuristic
// combination via synthetic [*node.Struct] values. The fixture in
// the demoproject only exercises realistic shapes; this test
// drives the contrived combinations that round out the override
// matrix without polluting the demo with coverage-only entities.
func TestOnStruct_OverrideMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		build        func() *node.Struct
		wantDetected bool
		wantMethod   string
	}{
		{
			name: "no override, no heuristic match",
			build: func() *node.Struct {
				return newStruct("pkg/x", "Plain")
			},
			wantDetected: false,
			wantMethod:   "",
		},
		{
			name: "no override, heuristic match",
			build: func() *node.Struct {
				return withWriteMethod(newStruct("pkg/x", "Logger"))
			},
			wantDetected: true,
			wantMethod:   "pkg/x.Logger.Write",
		},
		{
			name: "positive override, no heuristic match",
			build: func() *node.Struct {
				return withDirective(newStruct("pkg/x", "Forced"), DirectiveName, false)
			},
			wantDetected: true,
			wantMethod:   "",
		},
		{
			name: "positive override, heuristic match",
			build: func() *node.Struct {
				return withDirective(withWriteMethod(newStruct("pkg/x", "Both")), DirectiveName, false)
			},
			wantDetected: true,
			wantMethod:   "pkg/x.Both.Write",
		},
		{
			name: "negative override, no heuristic match",
			build: func() *node.Struct {
				return withDirective(newStruct("pkg/x", "Hidden"), DirectiveName, true)
			},
			wantDetected: false,
			wantMethod:   "",
		},
		{
			name: "negative override, heuristic match",
			build: func() *node.Struct {
				return withDirective(withWriteMethod(newStruct("pkg/x", "Silent")), DirectiveName, true)
			},
			wantDetected: false,
			wantMethod:   "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			s := tc.build()
			p.OnStruct(nil, s)
			gotDetected, _ := Detected.Get(s.Meta())
			if gotDetected != tc.wantDetected {
				t.Fatalf("detected = %v, want %v", gotDetected, tc.wantDetected)
			}
			gotMethod, _ := MethodQName.Get(s.Meta())
			if gotMethod != tc.wantMethod {
				t.Fatalf("method = %q, want %q", gotMethod, tc.wantMethod)
			}
		})
	}
}

// TestMatchSignature_RejectionBranches covers the heuristic's
// every-rejection-path invariant: a struct with a method named
// "Write" but mismatching parameter type, return arity, return
// type, or return builtin-ness must fall through to the no-match
// outcome. Drives every continue branch in matchSignature.
func TestMatchSignature_RejectionBranches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		method *node.Method
	}{
		{
			name: "wrong param type (string instead of []byte)",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "s", Type: builtin("string")}},
				Returns: []*node.TypeRef{builtin("int"), builtin("error")},
			},
		},
		{
			name: "short return arity",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "p", Type: byteSlice()}},
				Returns: []*node.TypeRef{builtin("error")},
			},
		},
		{
			name: "wrong return type (string instead of int)",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "p", Type: byteSlice()}},
				Returns: []*node.TypeRef{builtin("string"), builtin("error")},
			},
		},
		{
			name: "non-builtin return (pointer first slot)",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "p", Type: byteSlice()}},
				Returns: []*node.TypeRef{pointer(builtin("Article")), builtin("int")},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStruct("pkg/x", "Probe")
			s.Methods = []*node.Method{tc.method}
			if got, ok := matchSignature(s); ok {
				t.Fatalf("expected no match; got method=%v", got)
			}
		})
	}
}

// TestIsBuiltin_GuardClauses covers the defensive guard branches in
// isBuiltin — nil ref and non-builtin ref both short-circuit to
// false. The heuristic's exact-match logic depends on the guards
// rejecting before the Name comparison runs.
func TestIsBuiltin_GuardClauses(t *testing.T) {
	t.Parallel()

	t.Run("nil ref reaches the guard and returns false", func(t *testing.T) {
		t.Parallel()
		if isBuiltin(nil, "int") {
			t.Fatalf("nil ref should not match any builtin name")
		}
	})

	t.Run("non-builtin ref (pointer) returns false", func(t *testing.T) {
		t.Parallel()
		if isBuiltin(pointer(builtin("int")), "int") {
			t.Fatalf("pointer ref should not match builtin int")
		}
	})

	t.Run("non-builtin ref (qualified name) returns false", func(t *testing.T) {
		t.Parallel()
		if isBuiltin(&node.TypeRef{TypeKind: node.TypeRefNamed, Package: "fmt", Name: "Stringer"}, "Stringer") {
			t.Fatalf("qualified Named ref should not match an unqualified builtin")
		}
	})
}

// TestIsByteSlice_GuardClauses covers the [] byte detection
// guards: nil, non-slice, and slice-with-nil-elem all return
// false.
func TestIsByteSlice_GuardClauses(t *testing.T) {
	t.Parallel()

	t.Run("nil ref returns false", func(t *testing.T) {
		t.Parallel()
		if isByteSlice(nil) {
			t.Fatalf("nil ref should not be a byte slice")
		}
	})

	t.Run("non-slice ref returns false", func(t *testing.T) {
		t.Parallel()
		if isByteSlice(builtin("byte")) {
			t.Fatalf("non-slice ref should not be a byte slice")
		}
	})

	t.Run("slice with nil elem returns false", func(t *testing.T) {
		t.Parallel()
		if isByteSlice(&node.TypeRef{TypeKind: node.TypeRefSlice, Elem: nil}) {
			t.Fatalf("slice with nil elem should not be a byte slice")
		}
	})

	t.Run("uint8 slice satisfies the byte-slice predicate", func(t *testing.T) {
		t.Parallel()
		if !isByteSlice(&node.TypeRef{TypeKind: node.TypeRefSlice, Elem: builtin("uint8")}) {
			t.Fatalf("[]uint8 should be recognised as a byte slice")
		}
	})
}

// newStruct constructs a minimal Named struct in pkgPath with the
// supplied name. Used by the override-matrix table to keep each
// case's fixture concise.
func newStruct(pkgPath, name string) *node.Struct {
	return &node.Struct{Name: name, Package: pkgPath}
}

// withWriteMethod appends the canonical io.Writer-shaped method to
// s and returns s. The method's owner is left unset because the
// helpers under test do not depend on the owner pointer.
func withWriteMethod(s *node.Struct) *node.Struct {
	s.Methods = append(s.Methods, &node.Method{
		Name:    "Write",
		Params:  []*node.Param{{Name: "p", Type: byteSlice()}},
		Returns: []*node.TypeRef{builtin("int"), builtin("error")},
	})
	return s
}

// withDirective attaches a `+gen:<name>` directive (negated when
// neg is true) to s's directive list and returns s. The helper
// keeps the override-matrix test cases readable.
func withDirective(s *node.Struct, name directive.Name, neg bool) *node.Struct {
	s.DirectiveList = append(s.DirectiveList, &directive.Directive{Name: name, Negated: neg})
	return s
}

// builtin constructs an unqualified Named TypeRef for the given
// builtin name (e.g. `int`, `error`).
func builtin(name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Name: name}
}

// byteSlice constructs a fresh `[]byte` TypeRef.
func byteSlice() *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: builtin("byte")}
}

// pointer wraps elem in a Pointer TypeRef.
func pointer(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: elem}
}
