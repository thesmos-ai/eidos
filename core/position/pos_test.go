// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package position_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/position"
)

func TestAt(t *testing.T) {
	t.Parallel()
	got := position.At("a.go", 12, 5)
	want := position.Pos{File: "a.go", Line: 12, Column: 5}
	if got != want {
		t.Fatalf("At = %+v, want %+v", got, want)
	}
}

func TestAtOffset(t *testing.T) {
	t.Parallel()
	got := position.AtOffset("a.go", 12, 5, 99)
	want := position.Pos{File: "a.go", Line: 12, Column: 5, Offset: 99}
	if got != want {
		t.Fatalf("AtOffset = %+v, want %+v", got, want)
	}
}

func TestPos_IsZero(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		pos  position.Pos
		want bool
	}{
		{"zero value reports zero", position.Pos{}, true},
		{"only File set is non-zero", position.Pos{File: "a.go"}, false},
		{"only Line set is non-zero", position.Pos{Line: 1}, false},
		{"only Column set is non-zero", position.Pos{Column: 1}, false},
		{"only Offset set is non-zero", position.Pos{Offset: 1}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.pos.IsZero(); got != tc.want {
				t.Fatalf("IsZero(%+v) = %v, want %v", tc.pos, got, tc.want)
			}
		})
	}
}

func TestPos_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		pos  position.Pos
		want string
	}{
		{"zero renders as empty", position.Pos{}, ""},
		{"file only renders as file", position.Pos{File: "a.go"}, "a.go"},
		{"file and line render as file:line", position.At("a.go", 12, 0), "a.go:12"},
		{"file line and column render as file:line:col", position.At("a.go", 12, 5), "a.go:12:5"},
		{"offset is not surfaced", position.AtOffset("a.go", 12, 5, 99), "a.go:12:5"},
		{"column without line is suppressed", position.At("a.go", 0, 5), "a.go"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.pos.String(); got != tc.want {
				t.Fatalf("String(%+v) = %q, want %q", tc.pos, got, tc.want)
			}
		})
	}
}

func TestPos_Compare(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b position.Pos
		want int
	}{
		{
			"equal positions",
			position.AtOffset("a.go", 1, 2, 3), position.AtOffset("a.go", 1, 2, 3), 0,
		},
		{
			"file ordering: a.go before b.go",
			position.AtOffset("a.go", 99, 99, 99), position.AtOffset("b.go", 1, 1, 1), -1,
		},
		{
			"file ordering: b.go after a.go",
			position.AtOffset("b.go", 1, 1, 1), position.AtOffset("a.go", 99, 99, 99), 1,
		},
		{
			"line ordering within file",
			position.AtOffset("a.go", 1, 99, 99), position.AtOffset("a.go", 2, 1, 1), -1,
		},
		{
			"column ordering within line",
			position.AtOffset("a.go", 1, 1, 99), position.AtOffset("a.go", 1, 2, 1), -1,
		},
		{
			"offset ordering within column",
			position.AtOffset("a.go", 1, 1, 1), position.AtOffset("a.go", 1, 1, 2), -1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.a.Compare(tc.b); got != tc.want {
				t.Fatalf("Compare(%+v, %+v) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestPos_Equal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b position.Pos
		want bool
	}{
		{
			"identical positions are equal",
			position.AtOffset("a.go", 1, 2, 3), position.AtOffset("a.go", 1, 2, 3), true,
		},
		{
			"different file is not equal",
			position.AtOffset("a.go", 1, 2, 3), position.AtOffset("b.go", 1, 2, 3), false,
		},
		{
			"different line is not equal",
			position.AtOffset("a.go", 1, 2, 3), position.AtOffset("a.go", 2, 2, 3), false,
		},
		{
			"different column is not equal",
			position.AtOffset("a.go", 1, 2, 3), position.AtOffset("a.go", 1, 3, 3), false,
		},
		{
			"different offset is not equal",
			position.AtOffset("a.go", 1, 2, 3), position.AtOffset("a.go", 1, 2, 4), false,
		},
		{"two zero values are equal", position.Pos{}, position.Pos{}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.a.Equal(tc.b); got != tc.want {
				t.Fatalf("Equal(%+v, %+v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestPos_BeforeAfter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		a, b       position.Pos
		wantBefore bool
		wantAfter  bool
	}{
		{
			name: "earlier position is Before; later is After",
			a:    position.At("a.go", 1, 1), b: position.At("a.go", 5, 1),
			wantBefore: true, wantAfter: false,
		},
		{
			name: "later position is After; earlier is not",
			a:    position.At("a.go", 5, 1), b: position.At("a.go", 1, 1),
			wantBefore: false, wantAfter: true,
		},
		{
			name: "equal positions are neither before nor after",
			a:    position.At("a.go", 5, 1), b: position.At("a.go", 5, 1),
			wantBefore: false, wantAfter: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.a.Before(tc.b); got != tc.wantBefore {
				t.Fatalf("Before = %v, want %v", got, tc.wantBefore)
			}
			if got := tc.a.After(tc.b); got != tc.wantAfter {
				t.Fatalf("After = %v, want %v", got, tc.wantAfter)
			}
		})
	}
}

func TestSynthetic(t *testing.T) {
	t.Parallel()

	t.Run("wraps tag in angle brackets", func(t *testing.T) {
		t.Parallel()
		got := position.Synthetic("repogen")
		if got.File != "<repogen>" {
			t.Fatalf("Synthetic(\"repogen\").File = %q, want %q", got.File, "<repogen>")
		}
	})

	t.Run("empty tag still produces a recognisable synthetic file", func(t *testing.T) {
		t.Parallel()
		got := position.Synthetic("")
		if got.File != "<>" {
			t.Fatalf("Synthetic(\"\").File = %q, want %q", got.File, "<>")
		}
	})

	t.Run("synthetic position has zero line/column/offset", func(t *testing.T) {
		t.Parallel()
		got := position.Synthetic("generated")
		if got.Line != 0 || got.Column != 0 || got.Offset != 0 {
			t.Fatalf("Synthetic produced non-zero coordinates: %+v", got)
		}
	})
}

func TestPos_IsSynthetic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		pos  position.Pos
		want bool
	}{
		{"Synthetic-built Pos is synthetic", position.Synthetic("repogen"), true},
		{"empty-tag Synthetic is still synthetic", position.Synthetic(""), true},
		{"plain file path is not synthetic", position.At("a.go", 1, 1), false},
		{"zero Pos is not synthetic (empty File)", position.Pos{}, false},
		{
			"file starting with < but not ending with > is not synthetic",
			position.Pos{File: "<unfinished", Line: 1, Column: 1},
			false,
		},
		{
			"file ending with > but not starting with < is not synthetic",
			position.Pos{File: "unfinished>", Line: 1, Column: 1},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.pos.IsSynthetic(); got != tc.want {
				t.Fatalf("IsSynthetic(%+v) = %v, want %v", tc.pos, got, tc.want)
			}
		})
	}
}
