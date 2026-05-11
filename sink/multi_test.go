// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/sink"
)

// failSink returns a fixed error from every Write call. Used by
// multi-sink tests to drive partial-failure paths.
type failSink struct{ err error }

func (f *failSink) Write(emit.Target, []byte) error { return f.err }

func TestNewMulti(t *testing.T) {
	t.Parallel()

	t.Run("captures the supplied sink list", func(t *testing.T) {
		t.Parallel()
		a, b := sink.NewMemory(), sink.NewMemory()
		m := sink.NewMulti(a, b)
		if got := m.Sinks(); len(got) != 2 || got[0] != a || got[1] != b {
			t.Fatalf("Sinks should preserve registration order; got %+v", got)
		}
	})

	t.Run("filters nil sinks out", func(t *testing.T) {
		t.Parallel()
		a := sink.NewMemory()
		m := sink.NewMulti(nil, a, nil)
		if got := m.Sinks(); len(got) != 1 || got[0] != a {
			t.Fatalf("nil sinks should be filtered; got %+v", got)
		}
	})
}

func TestMulti_Write(t *testing.T) {
	t.Parallel()

	t.Run("dispatches to every underlying sink in registration order", func(t *testing.T) {
		t.Parallel()
		a, b := sink.NewMemory(), sink.NewMemory()
		m := sink.NewMulti(a, b)
		assertNoError(t, m.Write(targetAt("d", "x.go"), []byte("body")))
		gotA, _ := a.Get(targetAt("d", "x.go"))
		gotB, _ := b.Get(targetAt("d", "x.go"))
		if string(gotA) != "body" || string(gotB) != "body" {
			t.Fatalf("each sink should receive the body; got a=%q b=%q", gotA, gotB)
		}
	})

	t.Run("continues past a failing sink and joins errors", func(t *testing.T) {
		t.Parallel()
		errA := errors.New("sink-a failed")
		errC := errors.New("sink-c failed")
		good := sink.NewMemory()
		m := sink.NewMulti(&failSink{err: errA}, good, &failSink{err: errC})
		err := m.Write(targetAt("d", "x.go"), []byte("body"))
		if err == nil {
			t.Fatalf("expected joined error from failing sinks")
		}
		if !errors.Is(err, errA) || !errors.Is(err, errC) {
			t.Fatalf("joined error should match every component; got %v", err)
		}
		if !strings.Contains(err.Error(), "sink-a") || !strings.Contains(err.Error(), "sink-c") {
			t.Fatalf("error message should reference every failing sink; got %q", err.Error())
		}
		// The good sink in the middle still received the body.
		got, _ := good.Get(targetAt("d", "x.go"))
		if string(got) != "body" {
			t.Fatalf("good sink should still receive body despite failures; got %q", got)
		}
	})

	t.Run("returns nil when every sink succeeds", func(t *testing.T) {
		t.Parallel()
		m := sink.NewMulti(sink.NewMemory(), sink.NewMemory())
		if err := m.Write(targetAt("d", "x.go"), []byte("body")); err != nil {
			t.Fatalf("expected nil error; got %v", err)
		}
	})
}
