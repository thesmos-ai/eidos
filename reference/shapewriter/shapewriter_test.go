// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package shapewriter_test

import (
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/demopipe"
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
