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
