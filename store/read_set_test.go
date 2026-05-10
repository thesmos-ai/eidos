// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"slices"
	"sync"
	"testing"

	"go.thesmos.sh/eidos/store"
)

func TestNewReadSet(t *testing.T) {
	t.Parallel()

	t.Run("returns an empty ReadSet", func(t *testing.T) {
		t.Parallel()
		r := store.NewReadSet()
		if r.Len() != 0 {
			t.Fatalf("new ReadSet should be empty; got Len=%d", r.Len())
		}
	})
}

func TestReadSet_Record(t *testing.T) {
	t.Parallel()

	t.Run("records a key", func(t *testing.T) {
		t.Parallel()
		r := store.NewReadSet()
		r.Record("key-a")
		if !r.Has("key-a") {
			t.Fatalf("Record should make Has return true")
		}
	})

	t.Run("recording the same key twice is idempotent", func(t *testing.T) {
		t.Parallel()
		r := store.NewReadSet()
		r.Record("key-a")
		r.Record("key-a")
		if r.Len() != 1 {
			t.Fatalf("Len after duplicate Record = %d, want 1", r.Len())
		}
	})
}

func TestReadSet_Has(t *testing.T) {
	t.Parallel()

	t.Run("returns false for unrecorded keys", func(t *testing.T) {
		t.Parallel()
		r := store.NewReadSet()
		if r.Has("missing") {
			t.Fatalf("Has on empty ReadSet should be false")
		}
	})
}

func TestReadSet_Len(t *testing.T) {
	t.Parallel()

	t.Run("counts distinct recorded keys", func(t *testing.T) {
		t.Parallel()
		r := store.NewReadSet()
		r.Record("a")
		r.Record("b")
		r.Record("c")
		if r.Len() != 3 {
			t.Fatalf("Len = %d, want 3", r.Len())
		}
	})
}

func TestReadSet_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns recorded keys in lexicographic order", func(t *testing.T) {
		t.Parallel()
		r := store.NewReadSet()
		r.Record("c")
		r.Record("a")
		r.Record("b")
		if !slices.Equal(r.Keys(), []string{"a", "b", "c"}) {
			t.Fatalf("Keys not sorted: %v", r.Keys())
		}
	})
}

func TestReadSet_Hash(t *testing.T) {
	t.Parallel()

	t.Run("identical keys produce identical hashes regardless of insertion order", func(t *testing.T) {
		t.Parallel()
		r1 := store.NewReadSet()
		r1.Record("a")
		r1.Record("b")
		r1.Record("c")
		r2 := store.NewReadSet()
		r2.Record("c")
		r2.Record("b")
		r2.Record("a")
		if r1.Hash() != r2.Hash() {
			t.Fatalf("hashes should be order-independent")
		}
	})

	t.Run("different keys produce different hashes", func(t *testing.T) {
		t.Parallel()
		r1 := store.NewReadSet()
		r1.Record("a")
		r2 := store.NewReadSet()
		r2.Record("b")
		if r1.Hash() == r2.Hash() {
			t.Fatalf("hashes should differ for different keys")
		}
	})

	t.Run("empty ReadSet has a deterministic empty hash", func(t *testing.T) {
		t.Parallel()
		r1 := store.NewReadSet()
		r2 := store.NewReadSet()
		if r1.Hash() != r2.Hash() {
			t.Fatalf("empty ReadSets should produce the same hash")
		}
	})
}

func TestReadSet_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("Record and Hash are safe under -race", func(t *testing.T) {
		t.Parallel()
		r := store.NewReadSet()
		var wg sync.WaitGroup
		for i := range 16 {
			wg.Go(func() {
				r.Record(string(rune('a' + i)))
			})
		}
		for range 4 {
			wg.Go(func() {
				_ = r.Hash()
			})
		}
		wg.Wait()
	})
}
