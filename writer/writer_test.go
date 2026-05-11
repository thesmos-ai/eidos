// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer_test

import (
	"sync"
	"testing"

	"go.thesmos.sh/eidos/writer"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a Writer with the supplied Target and a fresh ImportSet", func(t *testing.T) {
		t.Parallel()
		target := targetAt("internal/repo", "user_gen.go")
		w := writer.New(target, nil)
		if w.Target() != target {
			t.Fatalf("Target = %+v, want %+v", w.Target(), target)
		}
		if w.Imports() == nil {
			t.Fatalf("Imports should be non-nil")
		}
	})

	t.Run("passes the custom AliasFunc through to the ImportSet", func(t *testing.T) {
		t.Parallel()
		w := writer.New(targetAt("a", "b.go"), func(string) string { return "FIXED" })
		alias, err := w.Imports().Imp("context")
		assertNoError(t, err)
		if alias != "FIXED" {
			t.Fatalf("Writer should pass the AliasFunc through; got %q", alias)
		}
	})
}

func TestWriter_Append(t *testing.T) {
	t.Parallel()

	t.Run("appends bytes in order", func(t *testing.T) {
		t.Parallel()
		w := writer.New(targetAt("a", "b.go"), nil)
		w.Append([]byte("hello "))
		w.Append([]byte("world"))
		if string(w.Body()) != "hello world" {
			t.Fatalf("body mismatch: %q", w.Body())
		}
	})

	t.Run("AppendString writes string content", func(t *testing.T) {
		t.Parallel()
		w := writer.New(targetAt("a", "b.go"), nil)
		w.AppendString("hello")
		if string(w.Body()) != "hello" {
			t.Fatalf("body mismatch: %q", w.Body())
		}
	})

	t.Run("Body returns a defensive copy", func(t *testing.T) {
		t.Parallel()
		w := writer.New(targetAt("a", "b.go"), nil)
		w.AppendString("hello")
		body := w.Body()
		body[0] = 'X'
		if string(w.Body()) != "hello" {
			t.Fatalf("Body should return a defensive copy; sink mutated to %q", w.Body())
		}
	})

	t.Run("Len reports the buffer size", func(t *testing.T) {
		t.Parallel()
		w := writer.New(targetAt("a", "b.go"), nil)
		w.AppendString("hello")
		if w.Len() != 5 {
			t.Fatalf("Len = %d, want 5", w.Len())
		}
	})
}

func TestWriter_ConcurrentAppend(t *testing.T) {
	t.Parallel()

	t.Run("concurrent Append calls do not interleave bytes (each call is atomic)", func(t *testing.T) {
		t.Parallel()
		w := writer.New(targetAt("a", "b.go"), nil)
		var wg sync.WaitGroup
		for range 16 {
			wg.Go(func() {
				w.AppendString("x")
			})
		}
		wg.Wait()
		if w.Len() != 16 {
			t.Fatalf("expected 16 bytes after 16 concurrent appends; got %d", w.Len())
		}
	})
}

func TestWriter_AppendKeyed(t *testing.T) {
	t.Parallel()

	t.Run("keyed contributions sort alphabetically by key on finalisation", func(t *testing.T) {
		t.Parallel()
		w := writer.New(targetAt("a", "b.go"), nil)
		w.AppendKeyed("c", []byte("CCC"))
		w.AppendKeyed("a", []byte("AAA"))
		w.AppendKeyed("b", []byte("BBB"))
		if got := string(w.Body()); got != "AAABBBCCC" {
			t.Fatalf("keyed contributions should sort by key; got %q", got)
		}
	})

	t.Run("concurrent AppendKeyed produces a deterministic body", func(t *testing.T) {
		t.Parallel()
		// Two runs of the same workload should produce identical
		// bodies even though goroutine schedules differ.
		run := func() string {
			w := writer.New(targetAt("a", "b.go"), nil)
			var wg sync.WaitGroup
			for i := range 16 {
				wg.Go(func() {
					w.AppendKeyed(string(rune('a'+i)), []byte{byte('A' + i)})
				})
			}
			wg.Wait()
			return string(w.Body())
		}
		first := run()
		second := run()
		if first != second {
			t.Fatalf("concurrent keyed appends should be deterministic; got %q vs %q", first, second)
		}
		if first != "ABCDEFGHIJKLMNOP" {
			t.Fatalf("expected alphabetical assembly; got %q", first)
		}
	})

	t.Run("sequential contributions render before keyed contributions", func(t *testing.T) {
		t.Parallel()
		w := writer.New(targetAt("a", "b.go"), nil)
		w.AppendString("seq1 ")
		w.AppendKeyed("z", []byte("z "))
		w.AppendString("seq2 ")
		w.AppendKeyed("a", []byte("a"))
		if got := string(w.Body()); got != "seq1 seq2 a"+"z " {
			// Sequential first (in append order), then keyed (sorted): "seq1 seq2 az "
			t.Fatalf("mixed-mode order mismatch; got %q", got)
		}
	})
}
