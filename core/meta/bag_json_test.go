// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
)

func TestBag_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("empty bag emits an empty entries array", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		got, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}
		want := `{"entries":[]}`
		if string(got) != want {
			t.Fatalf("Marshal output = %s, want %s", got, want)
		}
	})

	t.Run("sorts entries by name then authority for deterministic output", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		bagJSONScratchKey.Set(b, true, "test")
		bagJSONNumKey.Set(b, 42, "test")

		first, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("first Marshal returned error: %v", err)
		}
		second, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("second Marshal returned error: %v", err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("Marshal output is non-deterministic:\n %s\n %s", first, second)
		}

		// Sorted output places "test.num" before "test.scratch".
		idxNum := bytes.Index(first, []byte("test.num"))
		idxScratch := bytes.Index(first, []byte("test.scratch"))
		if idxNum > idxScratch {
			t.Fatalf("entries should be sorted by name; got %s", first)
		}
	})

	t.Run("tombstones round-trip without value bytes", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		bagJSONScratchKey.Set(b, true, "test")
		bagJSONScratchKey.Tombstone(b, "test")

		raw, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}
		if !strings.Contains(string(raw), `"tombstone":true`) {
			t.Fatalf("expected tombstone marker in output; got %s", raw)
		}
	})

	t.Run("propagates encoding errors from values json.Marshal cannot serialize", func(t *testing.T) {
		t.Parallel()
		b := meta.NewBag()
		// Channel values cannot be JSON-encoded; this exercises the
		// marshal-failure path inside marshalEntryValue. Stamping a
		// channel via a typed key gives us a value whose static type
		// json.Marshal rejects.
		bagJSONChanKey.Set(b, make(chan int), "test")
		if _, err := json.Marshal(b); err == nil {
			t.Fatalf("expected error when a stored value cannot encode")
		}
	})

	t.Run("propagates encoding errors from malformed RawMessage entries", func(t *testing.T) {
		t.Parallel()
		// A [json.RawMessage] containing invalid JSON bytes encodes
		// successfully via marshalEntryValue (which emits the
		// RawMessage verbatim) but fails the outer json.Marshal of
		// the bag's wire form — that is the only path through which
		// the outer-Marshal error branch is reachable in practice.
		b := meta.NewBag()
		bagJSONRawKey.Set(b, json.RawMessage("not valid json"), "test")
		if _, err := json.Marshal(b); err == nil {
			t.Fatalf("expected outer Marshal to reject malformed RawMessage")
		}
	})
}

func TestBag_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("round-trips primitive values through the typed Key.Get path", func(t *testing.T) {
		t.Parallel()
		src := meta.NewBag()
		bagJSONScratchKey.Set(src, true, "test")
		bagJSONNumKey.Set(src, 42, "test")

		raw, err := json.Marshal(src)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}

		dst := meta.NewBag()
		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatalf("Unmarshal returned error: %v", err)
		}

		if got, ok := bagJSONScratchKey.Get(dst); !ok || !got {
			t.Fatalf("scratch key should round-trip true; got %v ok=%v", got, ok)
		}
		if got, ok := bagJSONNumKey.Get(dst); !ok || got != 42 {
			t.Fatalf("num key should round-trip 42; got %d ok=%v", got, ok)
		}
	})

	t.Run("tombstones round-trip and remain tombstoned", func(t *testing.T) {
		t.Parallel()
		src := meta.NewBag()
		bagJSONScratchKey.Tombstone(src, "test")

		raw, err := json.Marshal(src)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}

		dst := meta.NewBag()
		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatalf("Unmarshal returned error: %v", err)
		}
		if !dst.Tombstoned(bagJSONScratchKey.Name()) {
			t.Fatalf("tombstone should survive round-trip")
		}
	})

	t.Run("preserves authority and position information", func(t *testing.T) {
		t.Parallel()
		src := meta.NewBag()
		pos := position.At("user.go", 12, 4)
		bagJSONScratchKey.SetAt(src, true, meta.AuthorityDirective, "directive", pos)

		raw, err := json.Marshal(src)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}

		dst := meta.NewBag()
		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatalf("Unmarshal returned error: %v", err)
		}

		got, ok := dst.Winning(bagJSONScratchKey.Name())
		if !ok {
			t.Fatalf("Winning should report the entry; got none")
		}
		if got.Authority != meta.AuthorityDirective {
			t.Fatalf("authority lost in round-trip: got %v", got.Authority)
		}
		if got.SetBy != "directive" {
			t.Fatalf("setBy lost in round-trip: got %q", got.SetBy)
		}
		if !got.Pos.Equal(pos) {
			t.Fatalf("pos lost in round-trip: got %v want %v", got.Pos, pos)
		}
	})

	t.Run("replaces any prior entries in the receiver", func(t *testing.T) {
		t.Parallel()
		dst := meta.NewBag()
		bagJSONScratchKey.Set(dst, true, "test")

		src := meta.NewBag()
		bagJSONNumKey.Set(src, 7, "test")
		raw, err := json.Marshal(src)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}

		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatalf("Unmarshal returned error: %v", err)
		}

		if _, ok := bagJSONScratchKey.Get(dst); ok {
			t.Fatalf("prior entry should be replaced by Unmarshal")
		}
		if got, ok := bagJSONNumKey.Get(dst); !ok || got != 7 {
			t.Fatalf("new entry not present after Unmarshal; got %d ok=%v", got, ok)
		}
	})

	t.Run("rejects entries with out-of-range authority", func(t *testing.T) {
		t.Parallel()
		dst := meta.NewBag()
		raw := []byte(`{"entries":[{"name":"x","authority":99,"set_by":"t"}]}`)
		if err := json.Unmarshal(raw, dst); err == nil {
			t.Fatalf("expected error for out-of-range authority")
		}
	})

	t.Run("propagates underlying JSON parse errors", func(t *testing.T) {
		t.Parallel()
		dst := meta.NewBag()
		// Top-level JSON is valid (it's an array) but does not match
		// the bagJSON struct shape, so the inner json.Unmarshal call
		// inside Bag.UnmarshalJSON surfaces the type-mismatch error.
		if err := json.Unmarshal([]byte("[1, 2, 3]"), dst); err == nil {
			t.Fatalf("expected error for wrong-shape JSON")
		}
	})
}

func TestBag_RoundTripDeterministic(t *testing.T) {
	t.Parallel()

	t.Run("encode → decode → encode is byte-stable", func(t *testing.T) {
		t.Parallel()
		src := meta.NewBag()
		bagJSONScratchKey.Set(src, true, "test")
		bagJSONNumKey.Set(src, 42, "test")
		bagJSONStrKey.Set(src, "alpha", "test")

		first, err := json.Marshal(src)
		if err != nil {
			t.Fatalf("first Marshal returned error: %v", err)
		}

		mid := meta.NewBag()
		if errU := json.Unmarshal(first, mid); errU != nil {
			t.Fatalf("Unmarshal returned error: %v", errU)
		}

		second, errM := json.Marshal(mid)
		if errM != nil {
			t.Fatalf("second Marshal returned error: %v", errM)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("round-trip drifted:\n first:  %s\n second: %s", first, second)
		}
	})
}

func TestKey_GetDecodesRawMessageLazily(t *testing.T) {
	t.Parallel()

	t.Run("returns the typed value after Unmarshal stores RawMessage", func(t *testing.T) {
		t.Parallel()
		src := meta.NewBag()
		bagJSONNumKey.Set(src, 99, "test")

		raw, err := json.Marshal(src)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}
		dst := meta.NewBag()
		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatalf("Unmarshal returned error: %v", err)
		}
		got, ok := bagJSONNumKey.Get(dst)
		if !ok || got != 99 {
			t.Fatalf("expected 99 via lazy decode; got %d ok=%v", got, ok)
		}
	})

	t.Run("returns false when the raw bytes cannot decode to T", func(t *testing.T) {
		t.Parallel()
		// Construct a wire form carrying a JSON string under a name
		// bound to an int-typed key; the lazy decoder must fail
		// cleanly rather than panicking.
		dst := meta.NewBag()
		raw := []byte(`{"entries":[{"name":"test.broken","authority":0,"value":"not a number","set_by":"t"}]}`)
		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatalf("Unmarshal returned error: %v", err)
		}
		if _, ok := bagJSONBrokenKey.Get(dst); ok {
			t.Fatalf("decoder mismatch should return ok=false")
		}
	})
}
