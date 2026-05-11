// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import (
	"sync"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
)

// assertEqualString fails the test if got and want differ.
func assertEqualString(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("string mismatch:\n got:  %q\n want: %q", got, want)
	}
}

func TestCapture(t *testing.T) {
	t.Parallel()

	t.Run("returns a fresh sink that records diagnostics", func(t *testing.T) {
		t.Parallel()
		s := diag.Capture()
		if s == nil {
			t.Fatalf("Capture returned nil")
		}
		s.Errorf(position.At("a.go", 1, 1), "boom")
		got := s.Diagnostics()
		if len(got) != 1 || got[0].Message != "boom" {
			t.Fatalf("Capture sink did not record: %+v", got)
		}
		if !s.HasErrors() {
			t.Fatalf("Capture sink should reflect recorded errors")
		}
	})

	t.Run("returns an empty sink", func(t *testing.T) {
		t.Parallel()
		s := diag.Capture()
		if got := s.Diagnostics(); len(got) != 0 {
			t.Fatalf("Capture sink should start empty; got %d", len(got))
		}
		if s.HasErrors() {
			t.Fatalf("Capture sink should not report errors before any Append")
		}
	})

	t.Run("each call returns an independent sink", func(t *testing.T) {
		t.Parallel()
		a := diag.Capture()
		b := diag.Capture()
		a.Errorf(position.Pos{}, "only-a")
		if len(b.Diagnostics()) != 0 {
			t.Fatalf("Capture sinks must be independent; b leaked %+v", b.Diagnostics())
		}
	})
}

func TestDiscard(t *testing.T) {
	t.Parallel()

	t.Run("Append drops the diagnostic", func(t *testing.T) {
		t.Parallel()
		s := diag.Discard()
		s.Append(diag.Diag{Severity: diag.Error, Message: "ignored"})
		if got := s.Diagnostics(); len(got) != 0 {
			t.Fatalf("Discard.Append must drop diagnostics; got %+v", got)
		}
	})

	t.Run("formatted helpers all become no-ops", func(t *testing.T) {
		t.Parallel()
		s := diag.Discard()
		s.Errorf(position.At("a.go", 1, 1), "e")
		s.Warnf(position.At("a.go", 2, 1), "w")
		s.Infof(position.At("a.go", 3, 1), "i")
		s.Internalf(position.At("a.go", 4, 1), "int")
		if got := s.Diagnostics(); len(got) != 0 {
			t.Fatalf("Discard helpers must drop diagnostics; got %+v", got)
		}
	})

	t.Run("PluginSink obtained via For also drops", func(t *testing.T) {
		t.Parallel()
		s := diag.Discard()
		p := s.For("repogen")
		p.Errorf(position.Pos{}, "from plugin")
		p.Warnf(position.Pos{}, "from plugin")
		p.Infof(position.Pos{}, "from plugin")
		if got := s.Diagnostics(); len(got) != 0 {
			t.Fatalf("Discard via PluginSink must drop; got %+v", got)
		}
	})

	t.Run("HasErrors always returns false even after Errorf", func(t *testing.T) {
		t.Parallel()
		s := diag.Discard()
		s.Errorf(position.Pos{}, "boom")
		s.Internalf(position.Pos{}, "panic")
		if s.HasErrors() {
			t.Fatalf("Discard.HasErrors must be false; got true")
		}
	})

	t.Run("Count always returns zero", func(t *testing.T) {
		t.Parallel()
		s := diag.Discard()
		s.Errorf(position.Pos{}, "e")
		s.Warnf(position.Pos{}, "w")
		s.Infof(position.Pos{}, "i")
		for _, sev := range []diag.Severity{diag.Info, diag.Warn, diag.Error, diag.Internal} {
			if got := s.Count(sev); got != 0 {
				t.Fatalf("Discard.Count(%v) = %d, want 0", sev, got)
			}
		}
	})

	t.Run("Diagnostics returns nil after appends", func(t *testing.T) {
		t.Parallel()
		s := diag.Discard()
		s.Errorf(position.Pos{}, "e")
		if got := s.Diagnostics(); got != nil {
			t.Fatalf("Discard.Diagnostics must return nil; got %+v", got)
		}
	})

	t.Run("concurrent appends are race-free no-ops", func(t *testing.T) {
		t.Parallel()
		const writers = 16
		const perWriter = 64
		s := diag.Discard()
		var wg sync.WaitGroup
		wg.Add(writers)
		for i := range writers {
			go func() {
				defer wg.Done()
				for j := range perWriter {
					s.Errorf(position.Pos{}, "w%d-%d", i, j)
				}
			}()
		}
		wg.Wait()
		if got := s.Diagnostics(); len(got) != 0 {
			t.Fatalf("concurrent Discard appends should remain empty; got %d", len(got))
		}
	})

	t.Run("returns an independent sink per call", func(t *testing.T) {
		t.Parallel()
		a := diag.Discard()
		b := diag.Discard()
		if a == b {
			t.Fatalf("Discard must return distinct sinks per call")
		}
	})
}
