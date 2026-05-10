// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
)

func TestRequired(t *testing.T) {
	t.Parallel()

	t.Run("flips PositionalArg.Required to true", func(t *testing.T) {
		t.Parallel()
		var p directive.PositionalArg
		directive.Required()(&p)
		if !p.Required {
			t.Fatalf("Required option should set Required=true")
		}
	})
}

func TestOneOf(t *testing.T) {
	t.Parallel()

	t.Run("appends the supplied values to PositionalArg.OneOf", func(t *testing.T) {
		t.Parallel()
		var p directive.PositionalArg
		directive.OneOf("alpha", "beta")(&p)
		if !slices.Equal(p.OneOf, []string{"alpha", "beta"}) {
			t.Fatalf("OneOf = %v, want [alpha beta]", p.OneOf)
		}
	})
}

func TestDefault(t *testing.T) {
	t.Parallel()

	t.Run("sets PositionalArg.Default", func(t *testing.T) {
		t.Parallel()
		var p directive.PositionalArg
		directive.Default("fallback")(&p)
		if p.Default != "fallback" {
			t.Fatalf("Default = %q, want %q", p.Default, "fallback")
		}
	})
}

func TestDescribe(t *testing.T) {
	t.Parallel()

	t.Run("sets PositionalArg.Description", func(t *testing.T) {
		t.Parallel()
		var p directive.PositionalArg
		directive.Describe("the output mode")(&p)
		assertEqualString(t, p.Description, "the output mode")
	})
}
