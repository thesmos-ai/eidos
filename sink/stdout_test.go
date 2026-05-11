// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink_test

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/sink"
)

func TestNewStdout(t *testing.T) {
	t.Parallel()

	t.Run("returns a Stdout sink that writes to the supplied writer", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		s := sink.NewStdout(&buf)
		assertNoError(t, s.Write(targetAt("a", "b.go"), []byte("hello")))
		if !strings.Contains(buf.String(), "// === a/b.go ===") {
			t.Fatalf("output should include target header; got %q", buf.String())
		}
		if !strings.Contains(buf.String(), "hello") {
			t.Fatalf("output should include body; got %q", buf.String())
		}
	})
}

func TestStdout_Write(t *testing.T) {
	t.Parallel()

	t.Run("rejects empty Filename with ErrInvalidTarget", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		s := sink.NewStdout(&buf)
		err := s.Write(targetAt("a", ""), nil)
		if !errors.Is(err, sink.ErrInvalidTarget) {
			t.Fatalf("Write should return ErrInvalidTarget; got %v", err)
		}
	})

	t.Run("falls back to the bare filename when Dir is empty", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		s := sink.NewStdout(&buf)
		assertNoError(t, s.Write(emit.Target{Filename: "x.go"}, []byte("body")))
		if !strings.Contains(buf.String(), "// === x.go ===") {
			t.Fatalf("output should include filename-only header; got %q", buf.String())
		}
	})

	t.Run("propagates header-write errors", func(t *testing.T) {
		t.Parallel()
		want := errors.New("disk full")
		s := sink.NewStdout(&errWriter{err: want})
		if got := s.Write(targetAt("a", "b.go"), []byte("body")); !errors.Is(got, want) {
			t.Fatalf("Write should propagate header error; got %v", got)
		}
	})

	t.Run("propagates body-write errors", func(t *testing.T) {
		t.Parallel()
		// Use a writer that succeeds for the header but fails on the
		// next write — emulated by counting writes.
		w := &boundedWriter{succeed: 1}
		s := sink.NewStdout(w)
		if err := s.Write(targetAt("a", "b.go"), []byte("body")); err == nil {
			t.Fatalf("Write should propagate body error")
		}
	})

	t.Run("propagates trailing newline errors", func(t *testing.T) {
		t.Parallel()
		w := &boundedWriter{succeed: 2}
		s := sink.NewStdout(w)
		if err := s.Write(targetAt("a", "b.go"), []byte("body")); err == nil {
			t.Fatalf("Write should propagate trailing-newline error")
		}
	})
}

func TestStdout_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	t.Run("Write serialises under -race", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		s := sink.NewStdout(&buf)
		var wg sync.WaitGroup
		for i := range 16 {
			wg.Go(func() {
				_ = s.Write(targetAt("d", string(rune('a'+i%26))+".go"), []byte("x"))
			})
		}
		wg.Wait()
	})
}
