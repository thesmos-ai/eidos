// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package position_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/position"
)

func TestPos_IsZero(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		pos  position.Pos
		want bool
	}{
		{"zero value reports zero", position.Pos{}, true},
		{"only File set is non-zero", at("a.go", 0, 0, 0), false},
		{"only Line set is non-zero", at("", 1, 0, 0), false},
		{"only Column set is non-zero", at("", 0, 1, 0), false},
		{"only Offset set is non-zero", at("", 0, 0, 1), false},
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
		{"file only renders as file", at("a.go", 0, 0, 0), "a.go"},
		{"file and line render as file:line", at("a.go", 12, 0, 0), "a.go:12"},
		{"file line and column render as file:line:col", at("a.go", 12, 5, 0), "a.go:12:5"},
		{"offset is not surfaced", at("a.go", 12, 5, 99), "a.go:12:5"},
		{"column without line is suppressed", at("a.go", 0, 5, 0), "a.go"},
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
		{"equal positions", at("a.go", 1, 2, 3), at("a.go", 1, 2, 3), 0},
		{"file ordering: a.go before b.go", at("a.go", 99, 99, 99), at("b.go", 1, 1, 1), -1},
		{"file ordering: b.go after a.go", at("b.go", 1, 1, 1), at("a.go", 99, 99, 99), 1},
		{"line ordering within file", at("a.go", 1, 99, 99), at("a.go", 2, 1, 1), -1},
		{"column ordering within line", at("a.go", 1, 1, 99), at("a.go", 1, 2, 1), -1},
		{"offset ordering within column", at("a.go", 1, 1, 1), at("a.go", 1, 1, 2), -1},
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
		{"identical positions are equal", at("a.go", 1, 2, 3), at("a.go", 1, 2, 3), true},
		{"different file is not equal", at("a.go", 1, 2, 3), at("b.go", 1, 2, 3), false},
		{"different line is not equal", at("a.go", 1, 2, 3), at("a.go", 2, 2, 3), false},
		{"different column is not equal", at("a.go", 1, 2, 3), at("a.go", 1, 3, 3), false},
		{"different offset is not equal", at("a.go", 1, 2, 3), at("a.go", 1, 2, 4), false},
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
			a:    at("a.go", 1, 1, 0), b: at("a.go", 5, 1, 0),
			wantBefore: true, wantAfter: false,
		},
		{
			name: "later position is After; earlier is not",
			a:    at("a.go", 5, 1, 0), b: at("a.go", 1, 1, 0),
			wantBefore: false, wantAfter: true,
		},
		{
			name: "equal positions are neither before nor after",
			a:    at("a.go", 5, 1, 0), b: at("a.go", 5, 1, 0),
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
		{"plain file path is not synthetic", at("a.go", 1, 1, 0), false},
		{"zero Pos is not synthetic (empty File)", position.Pos{}, false},
		{"file starting with < but not ending with > is not synthetic", at("<unfinished", 1, 1, 0), false},
		{"file ending with > but not starting with < is not synthetic", at("unfinished>", 1, 1, 0), false},
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
