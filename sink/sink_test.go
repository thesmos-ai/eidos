// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/sink"
)

// recordingSink captures every Write so the interface contract can
// be exercised end-to-end without depending on a real implementation.
type recordingSink struct {
	target emit.Target
	body   []byte
	err    error
}

func (r *recordingSink) Write(t emit.Target, body []byte) error {
	r.target = t
	r.body = body
	return r.err
}

func TestSink_InterfaceContract(t *testing.T) {
	t.Parallel()

	t.Run("Write delivers target and body to the implementation", func(t *testing.T) {
		t.Parallel()
		var s sink.Sink = &recordingSink{}
		target := emit.Target{Dir: "x", Filename: "y.go", Package: "x"}
		assertNoError(t, s.Write(target, []byte("hello")))
		got := s.(*recordingSink)
		if got.target != target {
			t.Fatalf("target mismatch: %+v", got.target)
		}
		if string(got.body) != "hello" {
			t.Fatalf("body mismatch: %q", got.body)
		}
	})

	t.Run("Write errors propagate to the caller", func(t *testing.T) {
		t.Parallel()
		want := errors.New("disk full")
		var s sink.Sink = &recordingSink{err: want}
		got := s.Write(emit.Target{Dir: "x", Filename: "y.go"}, nil)
		if !errors.Is(got, want) {
			t.Fatalf("Write should propagate err; got %v", got)
		}
	})
}
