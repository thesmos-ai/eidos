// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/storefixture"
)

func TestDirective(t *testing.T) {
	t.Parallel()

	t.Run("produces a positive directive with no args by default", func(t *testing.T) {
		t.Parallel()
		d := storefixture.Directive("repo")
		if d.Name != "repo" {
			t.Fatalf("name wrong: %q", d.Name)
		}
		if d.Negated {
			t.Fatalf("default directive should not be negated")
		}
		if len(d.Args) != 0 {
			t.Fatalf("default Args should be empty: %+v", d.Args)
		}
		if d.KV == nil {
			t.Fatalf("KV should be initialised, not nil")
		}
	})

	t.Run("applies options left-to-right", func(t *testing.T) {
		t.Parallel()
		pos := position.At("user.go", 7, 1)
		d := storefixture.Directive(
			"mock",
			storefixture.Negated(),
			storefixture.Arg("UserRepo"),
			storefixture.KV("target", "first"),
			storefixture.KV("target", "second"),
			storefixture.At(pos),
		)
		if !d.Negated {
			t.Fatalf("Negated() should mark the directive negated")
		}
		if len(d.Args) != 1 || d.Args[0] != "UserRepo" {
			t.Fatalf("Args wrong: %+v", d.Args)
		}
		if d.KV["target"] != "second" {
			t.Fatalf("later KV should overwrite earlier; got %q", d.KV["target"])
		}
		if !d.Pos.Equal(pos) {
			t.Fatalf("position wrong: got %v, want %v", d.Pos, pos)
		}
	})
}
