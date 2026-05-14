// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shapewriter_test

import (
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/eidostest/demopipe"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/shapewriter"
)

// TestPluginShape covers the plugin metadata surface: name, role
// implementations, and directive declarations. Keeps the plugin's
// public contract pinned at PR time so accidental rename / drop
// surfaces immediately.
func TestPluginShape(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented identifier", func(t *testing.T) {
		t.Parallel()
		if got := shapewriter.New().Name(); got != shapewriter.Name {
			t.Fatalf("Name = %q, want %q", got, shapewriter.Name)
		}
	})

	t.Run("implements Annotator, CapabilityProvider, DirectiveProvider", func(t *testing.T) {
		t.Parallel()
		p := shapewriter.New()
		if _, ok := any(p).(plugin.Annotator); !ok {
			t.Fatalf("plugin must implement plugin.Annotator")
		}
		if _, ok := any(p).(plugin.CapabilityProvider); !ok {
			t.Fatalf("plugin must implement plugin.CapabilityProvider")
		}
		if _, ok := any(p).(plugin.DirectiveProvider); !ok {
			t.Fatalf("plugin must implement plugin.DirectiveProvider")
		}
	})

	t.Run("Directives returns the writer schema", func(t *testing.T) {
		t.Parallel()
		schemas := shapewriter.New().Directives()
		if len(schemas) != 1 {
			t.Fatalf("expected one schema; got %d", len(schemas))
		}
		if schemas[0].Name != shapewriter.DirectiveName {
			t.Fatalf("schema name = %q, want %q", schemas[0].Name, shapewriter.DirectiveName)
		}
		if !schemas[0].AllowNegated {
			t.Fatalf("schema must allow the negated form for opt-out support")
		}
	})
}

// TestDetectsCanonicalWriter covers the heuristic happy path
// against the demoproject fixture: LineWriter has the canonical
// io.Writer signature so the annotator stamps detected=true with
// the matched method's QName.
func TestDetectsCanonicalWriter(t *testing.T) {
	t.Parallel()

	t.Run("LineWriter is detected and the matched method recorded", func(t *testing.T) {
		t.Parallel()
		result := runFixture(t)
		s, ok := result.Store.Nodes().Structs().ByQName("example.com/demoproject/blog.LineWriter")
		if !ok {
			t.Fatalf("fixture did not surface LineWriter in the node store")
		}
		detected, _ := shapewriter.Detected.Get(s.Meta())
		if !detected {
			t.Fatalf("LineWriter should be detected; got detected=false")
		}
		method, _ := shapewriter.MethodQName.Get(s.Meta())
		want := "example.com/demoproject/blog.LineWriter.Write"
		if method != want {
			t.Fatalf("method QName = %q, want %q", method, want)
		}
	})
}

// TestDoesNotDetectNonWriter covers the negative heuristic path:
// the fixture's Article struct has no Write method, so the
// annotator stamps detected=false plus an empty method back-link.
func TestDoesNotDetectNonWriter(t *testing.T) {
	t.Parallel()

	t.Run("Article reaches detected=false with no method back-link", func(t *testing.T) {
		t.Parallel()
		result := runFixture(t)
		s, ok := result.Store.Nodes().Structs().ByQName("example.com/demoproject/blog.Article")
		if !ok {
			t.Fatalf("fixture did not surface Article in the node store")
		}
		detected, _ := shapewriter.Detected.Get(s.Meta())
		if detected {
			t.Fatalf("Article should not be detected; got detected=true")
		}
		method, _ := shapewriter.MethodQName.Get(s.Meta())
		if method != "" {
			t.Fatalf("Article should carry an empty method back-link; got %q", method)
		}
	})
}

// TestRunDoesNotProduceDiagnostics covers the no-side-effects
// invariant: running the annotator over the fixture does not
// surface any error or warning diagnostics.
func TestRunDoesNotProduceDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("clean fixture produces no error or warn diagnostics", func(t *testing.T) {
		t.Parallel()
		result := runFixture(t)
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
	})
}

// runFixture runs the demopipe harness with the shape-writer
// annotator engaged against the demoproject fixture. Centralised
// so the per-aspect tests share a single fixture run path.
func runFixture(t *testing.T) demopipe.Result {
	t.Helper()
	return demopipe.Run(t, demopipe.RunOptions{
		Annotators: []plugin.Annotator{shapewriter.New()},
		Backend:    backend_golang.New(),
	})
}

// TestOnStructOverrideMatrix drives [shapewriter.Plugin.OnStruct]
// against every combination of {heuristic match, no match} ×
// {no directive, +gen:writer, -gen:writer} using synthetic struct
// values. Direct invocation of the exported hook covers the
// override logic without requiring contrived demoproject entities.
func TestOnStructOverrideMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		build        func() *node.Struct
		wantDetected bool
		wantMethod   string
	}{
		{
			name:         "no override, no heuristic match",
			build:        func() *node.Struct { return newStruct("pkg/x", "Plain") },
			wantDetected: false,
			wantMethod:   "",
		},
		{
			name:         "no override, heuristic match",
			build:        func() *node.Struct { return withWriteMethod(newStruct("pkg/x", "Logger")) },
			wantDetected: true,
			wantMethod:   "pkg/x.Logger.Write",
		},
		{
			name: "positive override, no heuristic match",
			build: func() *node.Struct {
				return withDirective(newStruct("pkg/x", "Forced"), false)
			},
			wantDetected: true,
			wantMethod:   "",
		},
		{
			name: "positive override, heuristic match",
			build: func() *node.Struct {
				return withDirective(withWriteMethod(newStruct("pkg/x", "Both")), false)
			},
			wantDetected: true,
			wantMethod:   "pkg/x.Both.Write",
		},
		{
			name: "negative override, no heuristic match",
			build: func() *node.Struct {
				return withDirective(newStruct("pkg/x", "Hidden"), true)
			},
			wantDetected: false,
			wantMethod:   "",
		},
		{
			name: "negative override, heuristic match",
			build: func() *node.Struct {
				return withDirective(withWriteMethod(newStruct("pkg/x", "Silent")), true)
			},
			wantDetected: false,
			wantMethod:   "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := tc.build()
			shapewriter.New().OnStruct(nil, s)
			detected, _ := shapewriter.Detected.Get(s.Meta())
			if detected != tc.wantDetected {
				t.Fatalf("detected = %v, want %v", detected, tc.wantDetected)
			}
			method, _ := shapewriter.MethodQName.Get(s.Meta())
			if method != tc.wantMethod {
				t.Fatalf("method = %q, want %q", method, tc.wantMethod)
			}
		})
	}
}

// TestOnStructSignatureRejection covers the heuristic's exact-
// signature invariant: methods named `Write` with mismatched
// parameter type, return arity, return types, or return builtin-
// ness all fall through to detected=false. Drives every rejection
// branch in the signature matcher through the exported hook.
func TestOnStructSignatureRejection(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		method *node.Method
	}{
		{
			name: "wrong param type (string instead of []byte)",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "s", Type: builtinRef("string")}},
				Returns: []*node.TypeRef{builtinRef("int"), builtinRef("error")},
			},
		},
		{
			name: "non-slice param (byte instead of []byte)",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "b", Type: builtinRef("byte")}},
				Returns: []*node.TypeRef{builtinRef("int"), builtinRef("error")},
			},
		},
		{
			name: "short return arity",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "p", Type: byteSliceRef()}},
				Returns: []*node.TypeRef{builtinRef("error")},
			},
		},
		{
			name: "wrong return type (string instead of int)",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "p", Type: byteSliceRef()}},
				Returns: []*node.TypeRef{builtinRef("string"), builtinRef("error")},
			},
		},
		{
			name: "non-builtin return (pointer first slot)",
			method: &node.Method{
				Name:    "Write",
				Params:  []*node.Param{{Name: "p", Type: byteSliceRef()}},
				Returns: []*node.TypeRef{pointerRef(builtinRef("Article")), builtinRef("int")},
			},
		},
		{
			name: "wrong method name",
			method: &node.Method{
				Name:    "WriteString",
				Params:  []*node.Param{{Name: "p", Type: byteSliceRef()}},
				Returns: []*node.TypeRef{builtinRef("int"), builtinRef("error")},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &node.Struct{Name: "Probe", Package: "pkg/x", Methods: []*node.Method{tc.method}}
			shapewriter.New().OnStruct(nil, s)
			detected, _ := shapewriter.Detected.Get(s.Meta())
			if detected {
				t.Fatalf("near-miss signature should reach detected=false; got true")
			}
		})
	}
}

// TestOnStructUint8Slice pins the byte-alias path: a method whose
// parameter is `[]uint8` (the type alias for byte) still matches
// the writer shape. Mirrors what gofmt-stripped Go source can
// produce when the alias is used explicitly.
func TestOnStructUint8Slice(t *testing.T) {
	t.Parallel()

	t.Run("uint8 slice matches the heuristic", func(t *testing.T) {
		t.Parallel()
		uint8Slice := &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: builtinRef("uint8")}
		s := &node.Struct{
			Name: "AliasWriter", Package: "pkg/x",
			Methods: []*node.Method{{
				Name:    "Write",
				Params:  []*node.Param{{Name: "p", Type: uint8Slice}},
				Returns: []*node.TypeRef{builtinRef("int"), builtinRef("error")},
			}},
		}
		shapewriter.New().OnStruct(nil, s)
		detected, _ := shapewriter.Detected.Get(s.Meta())
		if !detected {
			t.Fatalf("uint8 slice should match writer shape; got detected=false")
		}
	})
}

// newStruct constructs a minimal Named struct in pkgPath with the
// supplied name. Used by the synthetic-input tests to keep each
// case's fixture concise.
func newStruct(pkgPath, name string) *node.Struct {
	return &node.Struct{Name: name, Package: pkgPath}
}

// withWriteMethod appends the canonical io.Writer-shaped method to
// s and returns s. The method's owner is left unset because the
// hook does not depend on the owner pointer for the writer-shape
// match.
func withWriteMethod(s *node.Struct) *node.Struct {
	s.Methods = append(s.Methods, &node.Method{
		Name:    "Write",
		Params:  []*node.Param{{Name: "p", Type: byteSliceRef()}},
		Returns: []*node.TypeRef{builtinRef("int"), builtinRef("error")},
	})
	return s
}

// withDirective attaches a `+gen:writer` directive (negated when
// neg is true) to s's directive list and returns s.
func withDirective(s *node.Struct, neg bool) *node.Struct {
	s.DirectiveList = append(s.DirectiveList, &directive.Directive{
		Name:    shapewriter.DirectiveName,
		Negated: neg,
	})
	return s
}

// builtinRef constructs an unqualified Named TypeRef for the given
// builtin name.
func builtinRef(name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Name: name}
}

// byteSliceRef constructs a fresh `[]byte` TypeRef.
func byteSliceRef() *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: builtinRef("byte")}
}

// pointerRef wraps elem in a Pointer TypeRef.
func pointerRef(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: elem}
}

// TestConformance runs the framework's plugin-conformance suite
// against this package's plugin. The suite pins the standard
// framework contracts (stable Name, role-interface compliance,
// deterministic capability ordering, unique directive schema
// names, non-empty Versioned version) so a regression on any
// of them surfaces here before downstream tests trip over it.
func TestConformance(t *testing.T) {
	t.Parallel()
	plugintest.RunSuite(t, shapewriter.New())
}
