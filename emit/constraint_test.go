// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func TestConstraint_IsAny(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver reports true", func(t *testing.T) {
		t.Parallel()
		var c *emit.Constraint
		if !c.IsAny() {
			t.Fatalf("nil Constraint should report IsAny=true")
		}
	})

	t.Run("empty Embedded reports true", func(t *testing.T) {
		t.Parallel()
		c := &emit.Constraint{Raw: "any"}
		if !c.IsAny() {
			t.Fatalf("empty Constraint should report IsAny=true")
		}
	})

	t.Run("any Embedded entry makes IsAny false", func(t *testing.T) {
		t.Parallel()
		c := &emit.Constraint{Embedded: []emit.Ref{emit.Builtin("comparable")}}
		if c.IsAny() {
			t.Fatalf("Constraint with Embedded should report IsAny=false")
		}
	})
}

func TestConstraint_IsComparable(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver reports false", func(t *testing.T) {
		t.Parallel()
		var c *emit.Constraint
		if c.IsComparable() {
			t.Fatalf("nil Constraint should report IsComparable=false")
		}
	})

	t.Run("comparable BuiltinRef embedded reports true", func(t *testing.T) {
		t.Parallel()
		c := &emit.Constraint{Embedded: []emit.Ref{emit.Builtin("comparable")}}
		if !c.IsComparable() {
			t.Fatalf("comparable Embedded should report true")
		}
	})

	t.Run("non-comparable BuiltinRef embedded reports false", func(t *testing.T) {
		t.Parallel()
		c := &emit.Constraint{Embedded: []emit.Ref{emit.Builtin("any")}}
		if c.IsComparable() {
			t.Fatalf("non-comparable Embedded should report false")
		}
	})

	t.Run("ExternalRef named comparable is not the predeclared comparable", func(t *testing.T) {
		t.Parallel()
		c := &emit.Constraint{Embedded: []emit.Ref{emit.External("other/pkg", "comparable")}}
		if c.IsComparable() {
			t.Fatalf("external 'comparable' must not match the predeclared identifier")
		}
	})
}
