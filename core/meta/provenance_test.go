// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
)

func TestProvenance_String(t *testing.T) {
	t.Parallel()

	t.Run("renders set operation with value, plugin, and position", func(t *testing.T) {
		t.Parallel()
		p := meta.Provenance{
			Authority: meta.AuthorityPlugin,
			Tombstone: false,
			Value:     true,
			SetBy:     "writershape",
			Pos:       position.At("a.go", 42, 1),
		}
		got := p.String()
		for _, want := range []string{"plugin", "set true", "by writershape", "at a.go:42:1"} {
			if !strings.Contains(got, want) {
				t.Fatalf("String %q missing %q", got, want)
			}
		}
	})

	t.Run("renders tombstone without a value", func(t *testing.T) {
		t.Parallel()
		p := meta.Provenance{
			Authority: meta.AuthorityDirective,
			Tombstone: true,
			SetBy:     "directive",
			Pos:       position.At("a.go", 18, 1),
		}
		got := p.String()
		if !strings.Contains(got, "tombstone") {
			t.Fatalf("String %q should contain tombstone", got)
		}
		if strings.Contains(got, "set ") {
			t.Fatalf("String %q should not contain 'set ' for a tombstone", got)
		}
	})

	t.Run("renders CoveredBy when entry came via prefix", func(t *testing.T) {
		t.Parallel()
		p := meta.Provenance{
			Authority: meta.AuthorityDirective,
			Tombstone: true,
			SetBy:     "directive",
			CoveredBy: "shape.writer",
		}
		got := p.String()
		if !strings.Contains(got, "covers via shape.writer") {
			t.Fatalf("String %q should name the covering prefix", got)
		}
	})

	t.Run("omits SetBy and Pos when empty/zero", func(t *testing.T) {
		t.Parallel()
		p := meta.Provenance{Authority: meta.AuthorityPlugin, Value: "x"}
		got := p.String()
		if strings.Contains(got, "by ") {
			t.Fatalf("String %q should not say 'by' when SetBy is empty", got)
		}
		if strings.Contains(got, "at ") {
			t.Fatalf("String %q should not say 'at' when Pos is zero", got)
		}
	})
}
