// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package naming_test

import (
	"errors"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/naming"
)

func TestDefault(t *testing.T) {
	t.Parallel()

	t.Run("recognises CommonInitialisms", func(t *testing.T) {
		t.Parallel()
		got := naming.Default().Initialisms()
		want := slices.Clone(naming.CommonInitialisms)
		slices.Sort(want)
		assertEqualSlices(t, got, want)
	})

	t.Run("returns the same instance across calls", func(t *testing.T) {
		t.Parallel()
		first := naming.Default()
		second := naming.Default()
		if first != second {
			t.Fatalf("Default() should return a singleton; got two different pointers")
		}
	})
}

// TestCommonInitialisms_AreValid asserts every entry in the canonical
// list survives the same validation that user-supplied initialisms
// undergo. This guards Default's unchecked construction path.
func TestCommonInitialisms_AreValid(t *testing.T) {
	t.Parallel()
	if _, err := naming.New().WithInitialisms(naming.CommonInitialisms...); err != nil {
		t.Fatalf("CommonInitialisms contains an invalid entry: %v", err)
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("starts with no initialisms", func(t *testing.T) {
		t.Parallel()
		assertEqualSlices(t, naming.New().Initialisms(), nil)
	})
}

func TestCaser_WithInitialisms(t *testing.T) {
	t.Parallel()

	t.Run("appends the supplied initialisms", func(t *testing.T) {
		t.Parallel()
		c, err := naming.New().WithInitialisms("URL", "ID")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEqualSlices(t, c.Initialisms(), []string{"ID", "URL"})
	})

	t.Run("does not mutate the receiver", func(t *testing.T) {
		t.Parallel()
		base := naming.New()
		_, err := base.WithInitialisms("URL")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := base.Initialisms(); len(got) != 0 {
			t.Fatalf("receiver mutated: %v", got)
		}
	})

	t.Run("zero-value Caser is recovered into a usable Caser", func(t *testing.T) {
		t.Parallel()
		var zero naming.Caser
		c, err := zero.WithInitialisms("URL")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEqualSlices(t, c.Initialisms(), []string{"URL"})
	})

	t.Run("rejects invalid initialisms", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name string
			in   string
		}{
			{"empty string", ""},
			{"contains lower-case letter", "Url"},
			{"starts with a digit", "8K"},
			{"contains non-letter non-digit", "URL-"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := naming.New().WithInitialisms(tc.in)
				if !errors.Is(err, naming.ErrInvalidInitialism) {
					t.Fatalf("WithInitialisms(%q) error = %v, want ErrInvalidInitialism", tc.in, err)
				}
			})
		}
	})
}

func TestCaser_Initialisms(t *testing.T) {
	t.Parallel()

	t.Run("returns a sorted copy", func(t *testing.T) {
		t.Parallel()
		c, err := naming.New().WithInitialisms("URL", "ID", "API")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := c.Initialisms()
		assertEqualSlices(t, got, []string{"API", "ID", "URL"})
	})

	t.Run("mutating the returned slice does not affect the Caser", func(t *testing.T) {
		t.Parallel()
		c, err := naming.New().WithInitialisms("URL")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := c.Initialisms()
		got[0] = "MUTATED"
		if c.Initialisms()[0] != "URL" {
			t.Fatalf("returned slice aliased internal state")
		}
	})
}
