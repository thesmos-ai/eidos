// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package position_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/position"
)

func TestRange_IsZero(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		r    position.Range
		want bool
	}{
		{"zero value reports zero", position.Range{}, true},
		{"non-zero Start reports non-zero", position.Range{Start: at("a.go", 1, 1, 0)}, false},
		{"non-zero End reports non-zero", position.Range{End: at("a.go", 1, 1, 0)}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.r.IsZero(); got != tc.want {
				t.Fatalf("IsZero(%+v) = %v, want %v", tc.r, got, tc.want)
			}
		})
	}
}

func TestRange_Contains(t *testing.T) {
	t.Parallel()

	r := position.Range{
		Start: at("a.go", 1, 5, 10),
		End:   at("a.go", 3, 1, 50),
	}
	var zero position.Range

	cases := []struct {
		name string
		r    position.Range
		p    position.Pos
		want bool
	}{
		{"zero range contains nothing", zero, at("a.go", 1, 1, 0), false},
		{"different file is not contained", r, at("b.go", 1, 5, 10), false},
		{"position equal to Start is contained", r, at("a.go", 1, 5, 10), true},
		{"position strictly inside is contained", r, at("a.go", 2, 1, 25), true},
		{"position before Start is not contained", r, at("a.go", 1, 1, 5), false},
		{"position equal to End is not contained", r, at("a.go", 3, 1, 50), false},
		{"position after End is not contained", r, at("a.go", 5, 1, 100), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.r.Contains(tc.p); got != tc.want {
				t.Fatalf("Contains(%+v) on %+v = %v, want %v", tc.p, tc.r, got, tc.want)
			}
		})
	}
}

func TestRange_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		r    position.Range
		want string
	}{
		{"zero range renders as empty", position.Range{}, ""},
		{"collapsed range renders as a single position", position.Range{
			Start: at("a.go", 12, 5, 0),
			End:   at("a.go", 12, 5, 0),
		}, "a.go:12:5"},
		{"non-collapsed range renders as start-end", position.Range{
			Start: at("a.go", 1, 5, 0),
			End:   at("a.go", 3, 1, 0),
		}, "a.go:1:5-a.go:3:1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.r.String(); got != tc.want {
				t.Fatalf("String(%+v) = %q, want %q", tc.r, got, tc.want)
			}
		})
	}
}

func TestNewRange(t *testing.T) {
	t.Parallel()

	t.Run("returns a Range when start <= end and same file", func(t *testing.T) {
		t.Parallel()
		got, err := position.NewRange(at("a.go", 1, 1, 0), at("a.go", 5, 1, 0))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := position.Range{
			Start: at("a.go", 1, 1, 0),
			End:   at("a.go", 5, 1, 0),
		}
		if got != want {
			t.Fatalf("NewRange = %+v, want %+v", got, want)
		}
	})

	t.Run("collapsed range (start equal to end) is valid", func(t *testing.T) {
		t.Parallel()
		p := at("a.go", 1, 1, 0)
		got, err := position.NewRange(p, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Start != p || got.End != p {
			t.Fatalf("NewRange = %+v, want collapsed at %+v", got, p)
		}
	})

	t.Run("rejects cross-file ranges", func(t *testing.T) {
		t.Parallel()
		_, err := position.NewRange(at("a.go", 1, 1, 0), at("b.go", 1, 1, 0))
		if !errors.Is(err, position.ErrCrossFileRange) {
			t.Fatalf("NewRange cross-file error = %v, want ErrCrossFileRange", err)
		}
	})

	t.Run("rejects backwards ranges", func(t *testing.T) {
		t.Parallel()
		_, err := position.NewRange(at("a.go", 5, 1, 0), at("a.go", 1, 1, 0))
		if !errors.Is(err, position.ErrInvalidRangeOrder) {
			t.Fatalf("NewRange backwards error = %v, want ErrInvalidRangeOrder", err)
		}
	})
}

func TestRange_Union(t *testing.T) {
	t.Parallel()

	t.Run("union with zero is identity", func(t *testing.T) {
		t.Parallel()
		r := position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 5, 1, 0)}
		got, err := r.Union(position.Range{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != r {
			t.Fatalf("r.Union(zero) = %+v, want %+v", got, r)
		}
	})

	t.Run("zero union with non-zero returns the non-zero", func(t *testing.T) {
		t.Parallel()
		r := position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 5, 1, 0)}
		got, err := position.Range{}.Union(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != r {
			t.Fatalf("zero.Union(r) = %+v, want %+v", got, r)
		}
	})

	t.Run("two zero ranges union to zero", func(t *testing.T) {
		t.Parallel()
		got, err := position.Range{}.Union(position.Range{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.IsZero() {
			t.Fatalf("zero.Union(zero) = %+v, want zero", got)
		}
	})

	t.Run("disjoint ranges union to spanning range", func(t *testing.T) {
		t.Parallel()
		a := position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 2, 1, 0)}
		b := position.Range{Start: at("a.go", 5, 1, 0), End: at("a.go", 6, 1, 0)}
		got, err := a.Union(b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 6, 1, 0)}
		if got != want {
			t.Fatalf("Union = %+v, want %+v", got, want)
		}
	})

	t.Run("contained range unions to the wider range", func(t *testing.T) {
		t.Parallel()
		outer := position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 10, 1, 0)}
		inner := position.Range{Start: at("a.go", 4, 1, 0), End: at("a.go", 6, 1, 0)}
		got, err := outer.Union(inner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != outer {
			t.Fatalf("Union = %+v, want %+v", got, outer)
		}
	})

	t.Run("other starts earlier extends the union start", func(t *testing.T) {
		t.Parallel()
		later := position.Range{Start: at("a.go", 5, 1, 0), End: at("a.go", 10, 1, 0)}
		earlier := position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 3, 1, 0)}
		got, err := later.Union(earlier)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 10, 1, 0)}
		if got != want {
			t.Fatalf("Union = %+v, want %+v", got, want)
		}
	})

	t.Run("rejects cross-file union", func(t *testing.T) {
		t.Parallel()
		a := position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 2, 1, 0)}
		b := position.Range{Start: at("b.go", 1, 1, 0), End: at("b.go", 2, 1, 0)}
		_, err := a.Union(b)
		if !errors.Is(err, position.ErrCrossFileRange) {
			t.Fatalf("cross-file Union error = %v, want ErrCrossFileRange", err)
		}
	})
}

func TestRange_Overlaps(t *testing.T) {
	t.Parallel()

	r := position.Range{Start: at("a.go", 5, 1, 0), End: at("a.go", 10, 1, 0)}

	cases := []struct {
		name string
		a, b position.Range
		want bool
	}{
		{
			name: "fully contained range overlaps",
			a:    r,
			b:    position.Range{Start: at("a.go", 6, 1, 0), End: at("a.go", 8, 1, 0)},
			want: true,
		},
		{
			name: "partially overlapping start",
			a:    r,
			b:    position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 7, 1, 0)},
			want: true,
		},
		{
			name: "partially overlapping end",
			a:    r,
			b:    position.Range{Start: at("a.go", 8, 1, 0), End: at("a.go", 15, 1, 0)},
			want: true,
		},
		{
			name: "touching at end is not overlap (half-open)",
			a:    r,
			b:    position.Range{Start: at("a.go", 10, 1, 0), End: at("a.go", 15, 1, 0)},
			want: false,
		},
		{
			name: "fully disjoint after",
			a:    r,
			b:    position.Range{Start: at("a.go", 20, 1, 0), End: at("a.go", 30, 1, 0)},
			want: false,
		},
		{
			name: "fully disjoint before",
			a:    r,
			b:    position.Range{Start: at("a.go", 1, 1, 0), End: at("a.go", 3, 1, 0)},
			want: false,
		},
		{
			name: "different files do not overlap",
			a:    r,
			b:    position.Range{Start: at("b.go", 5, 1, 0), End: at("b.go", 15, 1, 0)},
			want: false,
		},
		{
			name: "zero range on left never overlaps",
			a:    position.Range{},
			b:    r,
			want: false,
		},
		{
			name: "zero range on right never overlaps",
			a:    r,
			b:    position.Range{},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.a.Overlaps(tc.b); got != tc.want {
				t.Fatalf("Overlaps(%+v, %+v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
