// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"errors"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	t.Run("returns an empty registry", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		if r == nil {
			t.Fatalf("NewRegistry returned nil")
		}
		if got := r.Names(); len(got) != 0 {
			t.Fatalf("new registry should have no names; got %v", got)
		}
	})
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("stores a schema and makes it lookup-able", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		s := directive.NewSchema("mock").Build()
		assertNoError(t, r.Register(s), "Register")
		got, ok := r.Lookup("mock")
		if !ok {
			t.Fatalf("Lookup should find the registered schema")
		}
		if got.Name != "mock" {
			t.Fatalf("Lookup returned %q, want mock", got.Name)
		}
	})

	t.Run("returns ErrSchemaConflict on duplicate Name", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		assertNoError(t, r.Register(directive.NewSchema("mock").Build()), "first Register")
		err := r.Register(directive.NewSchema("mock").Build())
		if !errors.Is(err, directive.ErrSchemaConflict) {
			t.Fatalf("err = %v, want ErrSchemaConflict", err)
		}
	})
}

func TestRegistry_Lookup(t *testing.T) {
	t.Parallel()

	t.Run("returns false for an unregistered name", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		if _, ok := r.Lookup("nothing"); ok {
			t.Fatalf("Lookup should be false for unregistered name")
		}
	})
}

func TestRegistry_Names(t *testing.T) {
	t.Parallel()

	t.Run("returns sorted names of registered schemas", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		assertNoError(t, r.Register(directive.NewSchema("zeta").Build()), "register zeta")
		assertNoError(t, r.Register(directive.NewSchema("alpha").Build()), "register alpha")
		assertNoError(t, r.Register(directive.NewSchema("mu").Build()), "register mu")
		got := r.Names()
		want := []directive.Name{"alpha", "mu", "zeta"}
		if !slices.Equal(got, want) {
			t.Fatalf("Names = %v, want %v", got, want)
		}
	})
}

func TestRegistry_Suggest(t *testing.T) {
	t.Parallel()

	t.Run("returns false when the registry is empty", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		if _, ok := r.Suggest("anything"); ok {
			t.Fatalf("Suggest on empty registry should return false")
		}
	})

	t.Run("returns the closest registered name for a near-miss typo", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		assertNoError(t, r.Register(directive.NewSchema("mock").Build()), "register mock")
		assertNoError(t, r.Register(directive.NewSchema("repo").Build()), "register repo")
		got, ok := r.Suggest("moc")
		if !ok {
			t.Fatalf("Suggest should match a near-miss")
		}
		if got != "mock" {
			t.Fatalf("Suggest = %q, want %q", got, "mock")
		}
	})

	t.Run("returns false when no registered name is plausibly close", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		assertNoError(t, r.Register(directive.NewSchema("mock").Build()), "register mock")
		// "completelyunrelated" is far further than the half-length
		// threshold from "mock"; expect no suggestion.
		if _, ok := r.Suggest("completelyunrelated"); ok {
			t.Fatalf("Suggest should reject distant candidates")
		}
	})
}
