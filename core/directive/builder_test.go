// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/kind"
)

func TestNewSchema(t *testing.T) {
	t.Parallel()

	t.Run("returns a builder whose Build defaults AllowNegated to true", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").Build()
		if s.Name != "mock" {
			t.Fatalf("Name = %q, want %q", s.Name, "mock")
		}
		if !s.AllowNegated {
			t.Fatalf("AllowNegated default should be true")
		}
	})
}

func TestFromSchema(t *testing.T) {
	t.Parallel()

	t.Run("wraps an existing schema for further chaining", func(t *testing.T) {
		t.Parallel()
		base := directive.Schema{Name: "mock", AllowNegated: false}
		got := directive.FromSchema(base).Describe("wraps").Build()
		if got.Name != "mock" || got.AllowNegated {
			t.Fatalf("FromSchema lost fields: %+v", got)
		}
		if got.Description != "wraps" {
			t.Fatalf("Describe should attach description; got %q", got.Description)
		}
	})
}

func TestSchemaBuilder_On(t *testing.T) {
	t.Parallel()

	t.Run("appends to AppliesTo", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").On("interface", "struct").Build()
		if !slices.Equal(s.AppliesTo, []kind.Kind{"interface", "struct"}) {
			t.Fatalf("AppliesTo = %v", s.AppliesTo)
		}
	})
}

func TestSchemaBuilder_Requires(t *testing.T) {
	t.Parallel()

	t.Run("appends to Requires", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").Requires("repo", "fixture").Build()
		if !slices.Equal(s.Requires, []directive.Name{"repo", "fixture"}) {
			t.Fatalf("Requires = %v", s.Requires)
		}
	})
}

func TestSchemaBuilder_ExclusiveWith(t *testing.T) {
	t.Parallel()

	t.Run("appends to MutuallyExclusiveWith", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").ExclusiveWith("skip").Build()
		if !slices.Equal(s.MutuallyExclusiveWith, []directive.Name{"skip"}) {
			t.Fatalf("MutuallyExclusiveWith = %v", s.MutuallyExclusiveWith)
		}
	})
}

func TestSchemaBuilder_RequiredKeys(t *testing.T) {
	t.Parallel()

	t.Run("appends to RequiredKeys", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").RequiredKeys("target", "out").Build()
		if !slices.Equal(s.RequiredKeys, []string{"target", "out"}) {
			t.Fatalf("RequiredKeys = %v", s.RequiredKeys)
		}
	})
}

func TestSchemaBuilder_AllowedKeys(t *testing.T) {
	t.Parallel()

	t.Run("appends to AllowedKeys", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").AllowedKeys("target").AllowedKeys("out").Build()
		if !slices.Equal(s.AllowedKeys, []string{"target", "out"}) {
			t.Fatalf("AllowedKeys = %v", s.AllowedKeys)
		}
	})
}

func TestSchemaBuilder_Positional(t *testing.T) {
	t.Parallel()

	t.Run("appends a positional arg with options applied", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").
			Positional("variant", directive.Required(), directive.OneOf("a", "b"), directive.Default("a"), directive.Describe("which")).
			Build()
		if len(s.PositionalArgs) != 1 {
			t.Fatalf("expected one positional arg; got %d", len(s.PositionalArgs))
		}
		got := s.PositionalArgs[0]
		if got.Name != "variant" || !got.Required || got.Default != "a" {
			t.Fatalf("positional = %+v", got)
		}
		if !slices.Equal(got.OneOf, []string{"a", "b"}) {
			t.Fatalf("OneOf = %v", got.OneOf)
		}
		assertEqualString(t, got.Description, "which")
	})
}

func TestSchemaBuilder_AllowExtraPositional(t *testing.T) {
	t.Parallel()

	t.Run("sets AllowExtraPositional", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").AllowExtraPositional().Build()
		if !s.AllowExtraPositional {
			t.Fatalf("AllowExtraPositional should be true")
		}
	})
}

func TestSchemaBuilder_DenyNegation(t *testing.T) {
	t.Parallel()

	t.Run("clears AllowNegated", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").DenyNegation().Build()
		if s.AllowNegated {
			t.Fatalf("AllowNegated should be false after DenyNegation")
		}
	})
}

func TestSchemaBuilder_Describe(t *testing.T) {
	t.Parallel()

	t.Run("sets Description", func(t *testing.T) {
		t.Parallel()
		s := directive.NewSchema("mock").Describe("generates mock implementations").Build()
		assertEqualString(t, s.Description, "generates mock implementations")
	})
}
