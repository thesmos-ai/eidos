// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink_test

import (
	"errors"
	"sync"
	"testing"

	"go.thesmos.sh/eidos/sink"
)

func TestNewMemory(t *testing.T) {
	t.Parallel()

	t.Run("returns an empty Memory sink", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		if m.Len() != 0 {
			t.Fatalf("new Memory should be empty; Len=%d", m.Len())
		}
	})
}

func TestMemory_Write(t *testing.T) {
	t.Parallel()

	t.Run("records the body under the supplied target", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		assertNoError(t, m.Write(targetAt("a", "b.go"), []byte("hello")))
		got, ok := m.Get(targetAt("a", "b.go"))
		if !ok || string(got) != "hello" {
			t.Fatalf("Get mismatch: %q ok=%v", got, ok)
		}
	})

	t.Run("subsequent writes overwrite the prior body", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		assertNoError(t, m.Write(targetAt("a", "b.go"), []byte("first")))
		assertNoError(t, m.Write(targetAt("a", "b.go"), []byte("second")))
		got, _ := m.Get(targetAt("a", "b.go"))
		if string(got) != "second" {
			t.Fatalf("overwrite mismatch: %q", got)
		}
	})

	t.Run("rejects empty Filename with ErrInvalidTarget", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		err := m.Write(targetAt("a", ""), nil)
		if !errors.Is(err, sink.ErrInvalidTarget) {
			t.Fatalf("Write should return ErrInvalidTarget; got %v", err)
		}
	})

	t.Run("returns a defensive copy from Get", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		assertNoError(t, m.Write(targetAt("a", "b.go"), []byte("hello")))
		got, _ := m.Get(targetAt("a", "b.go"))
		got[0] = 'X'
		stored, _ := m.Get(targetAt("a", "b.go"))
		if string(stored) != "hello" {
			t.Fatalf("Get should return a defensive copy; sink mutated to %q", stored)
		}
	})
}

func TestMemory_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns nil and false for missing targets", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		if got, ok := m.Get(targetAt("a", "b.go")); ok || got != nil {
			t.Fatalf("Get on missing target = %q ok=%v", got, ok)
		}
	})
}

func TestMemory_Files(t *testing.T) {
	t.Parallel()

	t.Run("returns a snapshot of every recorded entry", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		assertNoError(t, m.Write(targetAt("a", "x.go"), []byte("X")))
		assertNoError(t, m.Write(targetAt("b", "y.go"), []byte("Y")))
		got := m.Files()
		if len(got) != 2 {
			t.Fatalf("Files = %d entries, want 2", len(got))
		}
	})

	t.Run("snapshot is independent of subsequent writes", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		assertNoError(t, m.Write(targetAt("a", "x.go"), []byte("X")))
		snap := m.Files()
		assertNoError(t, m.Write(targetAt("a", "x.go"), []byte("Z")))
		if string(snap[targetAt("a", "x.go")]) != "X" {
			t.Fatalf("snapshot should not see later writes")
		}
	})
}

func TestMemory_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes every recorded entry", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		assertNoError(t, m.Write(targetAt("a", "x.go"), []byte("X")))
		m.Clear()
		if m.Len() != 0 {
			t.Fatalf("Clear should empty the sink; Len=%d", m.Len())
		}
	})
}

func TestMemory_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	t.Run("Write and Get are safe under -race", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMemory()
		var wg sync.WaitGroup
		for i := range 32 {
			wg.Go(func() {
				_ = m.Write(targetAt("d", string(rune('a'+i%26))+".go"), []byte("x"))
			})
		}
		for range 8 {
			wg.Go(func() {
				_ = m.Files()
			})
		}
		wg.Wait()
	})
}
