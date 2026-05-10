// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta_test

import (
	"errors"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/core/meta"
)

func TestBoolParser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		in      string
		want    bool
		wantErr bool
	}{
		{"empty input is true (bare +gen:foo bar form)", "", true, false},
		{"lowercase true", "true", true, false},
		{"uppercase TRUE", "TRUE", true, false},
		{"mixed-case True", "True", true, false},
		{"shorthand 1", "1", true, false},
		{"lowercase false", "false", false, false},
		{"uppercase FALSE", "FALSE", false, false},
		{"shorthand 0", "0", false, false},
		{"non-bool input returns ErrParse", "bogus", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := meta.BoolParser(tc.in)
			if tc.wantErr {
				if !errors.Is(err, meta.ErrParse) {
					t.Fatalf("BoolParser(%q) err = %v, want ErrParse", tc.in, err)
				}
				return
			}
			assertNoError(t, err, "BoolParser")
			if got != tc.want {
				t.Fatalf("BoolParser(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestStringParser(t *testing.T) {
	t.Parallel()

	t.Run("returns the raw input unchanged", func(t *testing.T) {
		t.Parallel()
		got, err := meta.StringParser("hello world")
		assertNoError(t, err, "StringParser")
		assertEqualString(t, got, "hello world")
	})
}

func TestIntParser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{"positive integer", "42", 42, false},
		{"negative integer", "-1", -1, false},
		{"zero", "0", 0, false},
		{"non-numeric returns ErrParse", "abc", 0, true},
		{"empty string returns ErrParse", "", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := meta.IntParser(tc.in)
			if tc.wantErr {
				if !errors.Is(err, meta.ErrParse) {
					t.Fatalf("IntParser(%q) err = %v, want ErrParse", tc.in, err)
				}
				return
			}
			assertNoError(t, err, "IntParser")
			if got != tc.want {
				t.Fatalf("IntParser(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestStringListParser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty input yields empty slice", "", []string{}},
		{"single element", "alpha", []string{"alpha"}},
		{"comma-separated elements", "alpha,beta,gamma", []string{"alpha", "beta", "gamma"}},
		{"preserves whitespace inside elements", " a, b ", []string{" a", " b "}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := meta.StringListParser(tc.in)
			assertNoError(t, err, "StringListParser")
			if !slices.Equal(got, tc.want) {
				t.Fatalf("StringListParser(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}
