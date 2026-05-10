// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import (
	"encoding/json"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
)

func TestDiag_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("fully-populated diagnostic round-trips", func(t *testing.T) {
		t.Parallel()
		original := diag.Diag{
			Severity: diag.Error,
			Plugin:   "validation",
			Pos:      position.At("auth.go", 18, 5),
			Message:  "field secret cannot be omitempty",
			Detail:   "type is non-nullable",
		}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var decoded diag.Diag
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unexpected error decoding: %v", err)
		}
		if decoded != original {
			t.Fatalf("round-trip mismatch:\n got:  %+v\n want: %+v", decoded, original)
		}
	})

	t.Run("empty plugin and zero pos are omitted from output", func(t *testing.T) {
		t.Parallel()
		d := diag.Diag{Severity: diag.Warn, Message: "general warning"}
		data, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(data)
		if strings.Contains(s, "plugin") {
			t.Fatalf("plugin field should be omitted when empty: %s", s)
		}
		if strings.Contains(s, "pos") {
			t.Fatalf("pos field should be omitted when zero: %s", s)
		}
	})

	t.Run("severity renders as a string", func(t *testing.T) {
		t.Parallel()
		d := diag.Diag{Severity: diag.Internal, Message: "boom"}
		data, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(data), `"severity":"internal"`) {
			t.Fatalf("expected severity rendered as string; got %s", data)
		}
	})
}
