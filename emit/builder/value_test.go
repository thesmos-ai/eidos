// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestTag covers the [builder.Tag] one-liner constructor for the
// flat *emit.Tag value type passed across cross-cutting boundaries.
func TestTag(t *testing.T) {
	t.Parallel()

	t.Run("Tag returns *emit.Tag with the supplied key and value", func(t *testing.T) {
		t.Parallel()
		got := builder.Tag("json", "name,omitempty")
		if got == nil {
			t.Fatalf("Tag returned nil")
		}
		if got.Key != "json" || got.Value != "name,omitempty" {
			t.Fatalf("Tag fields wrong; got %+v", got)
		}
	})
}

// TestApproxExact covers the [builder.Approx] / [builder.Exact]
// constructors for [emit.UnionTerm]. Approx sets the `~T` flag;
// Exact leaves it false.
func TestApproxExact(t *testing.T) {
	t.Parallel()

	t.Run("Approx stamps UnionTerm.Approx=true", func(t *testing.T) {
		t.Parallel()
		ref := emit.Builtin("int")
		ut := builder.Approx(ref)
		if !ut.Approx {
			t.Fatalf("Approx should set Approx=true; got %+v", ut)
		}
		if ut.Type != ref {
			t.Fatalf("Approx Type field not threaded")
		}
	})

	t.Run("Exact leaves UnionTerm.Approx=false", func(t *testing.T) {
		t.Parallel()
		ref := emit.Builtin("string")
		ut := builder.Exact(ref)
		if ut.Approx {
			t.Fatalf("Exact should leave Approx=false; got %+v", ut)
		}
		if ut.Type != ref {
			t.Fatalf("Exact Type field not threaded")
		}
	})

	t.Run("Approx and Exact compose into emit.Union terms", func(t *testing.T) {
		t.Parallel()
		u := emit.Union(
			builder.Approx(emit.Builtin("int")),
			builder.Exact(emit.Builtin("string")),
		)
		if u == nil || len(u.UnionTerms) != 2 {
			t.Fatalf("Union should carry both terms; got %+v", u)
		}
		if !u.UnionTerms[0].Approx || u.UnionTerms[1].Approx {
			t.Fatalf("term Approx flags wrong; got %+v", u.UnionTerms)
		}
	})
}

// TestConstraintConstructors covers [AnyConstraint],
// [ComparableConstraint], and [Constraint(...)] — the three named
// helpers for [*emit.Constraint] construction.
func TestConstraintConstructors(t *testing.T) {
	t.Parallel()

	t.Run("AnyConstraint returns nil (the implicit-any case)", func(t *testing.T) {
		t.Parallel()
		if c := builder.AnyConstraint(); c != nil {
			t.Fatalf("AnyConstraint should be nil; got %+v", c)
		}
	})

	t.Run("ComparableConstraint embeds the comparable builtin", func(t *testing.T) {
		t.Parallel()
		c := builder.ComparableConstraint()
		if c == nil || !c.IsComparable() {
			t.Fatalf("ComparableConstraint should be IsComparable; got %+v", c)
		}
	})

	t.Run("Constraint(refs...) populates Embedded", func(t *testing.T) {
		t.Parallel()
		c := builder.Constraint(emit.Builtin("comparable"), emit.External("fmt", "Stringer"))
		if c == nil || len(c.Embedded) != 2 {
			t.Fatalf("Constraint should embed both refs; got %+v", c)
		}
	})

	t.Run("Constraint() with zero refs returns the implicit-any case", func(t *testing.T) {
		t.Parallel()
		if c := builder.Constraint(); c != nil {
			t.Fatalf("Constraint() should equal AnyConstraint; got %+v", c)
		}
	})
}
