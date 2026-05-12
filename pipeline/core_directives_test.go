// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline_test

import (
	"testing"

	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/sink"
)

// TestCoreDirectives_OutRegistered pins the core-set contract: a
// freshly-built pipeline always exposes the `out` directive
// through its [directive.Registry], even when the caller supplies
// no schemas via [pipeline.Builder.WithDirective]. The
// registration is what enables source authors to annotate a node
// with `+gen:out filename.go` and have the Layout phase pick it
// up — without a registry entry, the directive parser rejects the
// comment with an unknown-directive error.
func TestCoreDirectives_OutRegistered(t *testing.T) {
	t.Parallel()

	t.Run("out directive is in the core set after Build", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		schema, ok := p.DirectiveRegistry().Lookup(pipeline.OutDirective)
		if !ok {
			t.Fatalf("OutDirective should be registered in the core set; got missing")
		}
		if schema.Name != pipeline.OutDirective {
			t.Fatalf("schema name = %q, want %q", schema.Name, pipeline.OutDirective)
		}
		if schema.AllowNegated {
			t.Fatalf("OutDirective should reject the negated form")
		}
	})

	t.Run("out directive declares exactly one positional argument", func(t *testing.T) {
		t.Parallel()
		p, err := pipeline.New().
			WithFrontend(&stubFE{name: "fe"}).
			WithBackend(&stubBE{name: "be"}).
			WithSink(sink.NewMemory()).
			Build()
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		schema, ok := p.DirectiveRegistry().Lookup(pipeline.OutDirective)
		if !ok {
			t.Fatalf("OutDirective missing from registry")
		}
		if got := len(schema.PositionalArgs); got != 1 {
			t.Fatalf("PositionalArgs = %d, want 1 (the filename arg)", got)
		}
	})
}
