// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import (
	"encoding/json"
	"errors"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
)

func TestSeverity_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		sev  diag.Severity
		want string
	}{
		{"Info", diag.Info, "info"},
		{"Warn", diag.Warn, "warn"},
		{"Error", diag.Error, "error"},
		{"Internal", diag.Internal, "internal"},
		{"unknown stringifies with a marker", diag.Severity(99), "severity(99)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.sev.String(), tc.want)
		})
	}
}

func TestParseSeverity(t *testing.T) {
	t.Parallel()

	t.Run("known levels parse round-trip", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			in   string
			want diag.Severity
		}{
			{"info", diag.Info},
			{"warn", diag.Warn},
			{"error", diag.Error},
			{"internal", diag.Internal},
		}
		for _, tc := range cases {
			t.Run(tc.in, func(t *testing.T) {
				t.Parallel()
				got, err := diag.ParseSeverity(tc.in)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tc.want {
					t.Fatalf("ParseSeverity(%q) = %v, want %v", tc.in, got, tc.want)
				}
			})
		}
	})

	t.Run("unknown input returns ErrUnknownSeverity", func(t *testing.T) {
		t.Parallel()
		_, err := diag.ParseSeverity("bogus")
		if !errors.Is(err, diag.ErrUnknownSeverity) {
			t.Fatalf("ParseSeverity(\"bogus\") err = %v, want ErrUnknownSeverity", err)
		}
	})
}

func TestSeverity_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		sev  diag.Severity
		want string
	}{
		{"Info", diag.Info, `"info"`},
		{"Warn", diag.Warn, `"warn"`},
		{"Error", diag.Error, `"error"`},
		{"Internal", diag.Internal, `"internal"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tc.sev)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertEqualString(t, string(data), tc.want)
		})
	}
}

func TestSeverity_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("known string round-trips", func(t *testing.T) {
		t.Parallel()
		var s diag.Severity
		if err := json.Unmarshal([]byte(`"warn"`), &s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != diag.Warn {
			t.Fatalf("UnmarshalJSON(\"warn\") = %v, want Warn", s)
		}
	})

	t.Run("unknown string returns ErrUnknownSeverity", func(t *testing.T) {
		t.Parallel()
		var s diag.Severity
		err := json.Unmarshal([]byte(`"bogus"`), &s)
		if !errors.Is(err, diag.ErrUnknownSeverity) {
			t.Fatalf("Unmarshal(\"bogus\") err = %v, want ErrUnknownSeverity", err)
		}
	})

	t.Run("non-string input returns a parse error", func(t *testing.T) {
		t.Parallel()
		var s diag.Severity
		err := json.Unmarshal([]byte(`42`), &s)
		if err == nil {
			t.Fatalf("expected error for non-string severity input")
		}
		if errors.Is(err, diag.ErrUnknownSeverity) {
			t.Fatalf("non-string error should not be ErrUnknownSeverity; got %v", err)
		}
	})
}
