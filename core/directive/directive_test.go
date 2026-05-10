// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/directive"
)

func TestDirective_Arg(t *testing.T) {
	t.Parallel()

	t.Run("returns the value at the given positional index", func(t *testing.T) {
		t.Parallel()
		d := &directive.Directive{Args: []string{"alpha", "beta", "gamma"}}
		assertEqualString(t, d.Arg(1), "beta")
	})

	t.Run("returns empty string for an index out of range", func(t *testing.T) {
		t.Parallel()
		d := &directive.Directive{Args: []string{"alpha"}}
		assertEqualString(t, d.Arg(5), "")
	})

	t.Run("returns empty string for a negative index", func(t *testing.T) {
		t.Parallel()
		d := &directive.Directive{Args: []string{"alpha"}}
		assertEqualString(t, d.Arg(-1), "")
	})
}

func TestDirective_HasKey(t *testing.T) {
	t.Parallel()

	t.Run("returns true when the key is present", func(t *testing.T) {
		t.Parallel()
		d := &directive.Directive{KV: map[string]string{"target": "Repo"}}
		if !d.HasKey("target") {
			t.Fatalf("HasKey should be true for present key")
		}
	})

	t.Run("returns false when the key is absent", func(t *testing.T) {
		t.Parallel()
		d := &directive.Directive{KV: map[string]string{}}
		if d.HasKey("absent") {
			t.Fatalf("HasKey should be false for absent key")
		}
	})
}

func TestDirective_Value(t *testing.T) {
	t.Parallel()

	t.Run("returns the value when present", func(t *testing.T) {
		t.Parallel()
		d := &directive.Directive{KV: map[string]string{"target": "Repo"}}
		assertEqualString(t, d.Value("target"), "Repo")
	})

	t.Run("returns empty string when absent", func(t *testing.T) {
		t.Parallel()
		d := &directive.Directive{KV: map[string]string{}}
		assertEqualString(t, d.Value("absent"), "")
	})
}
