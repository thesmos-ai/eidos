// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import (
	"strconv"
	"sync"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil sink with no diagnostics recorded", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		if s == nil {
			t.Fatalf("New returned nil")
		}
		if got := s.Diagnostics(); len(got) != 0 {
			t.Fatalf("New Sink should be empty; got %d diagnostics", len(got))
		}
	})
}

func TestSink_Append(t *testing.T) {
	t.Parallel()

	t.Run("records the diagnostic verbatim", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		d := diag.Diag{Severity: diag.Warn, Message: "first"}
		s.Append(d)
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != d {
			t.Fatalf("Append did not record diagnostic: got %+v", got)
		}
	})

	t.Run("concurrent appends are race-free and total-ordered", func(t *testing.T) {
		t.Parallel()
		const writers = 16
		const perWriter = 64
		s := diag.New()
		var wg sync.WaitGroup
		wg.Add(writers)
		for i := range writers {
			go func() {
				defer wg.Done()
				for j := range perWriter {
					s.Errorf(position.Pos{}, "w%d-%s", i, strconv.Itoa(j))
				}
			}()
		}
		wg.Wait()
		if got, want := len(s.Diagnostics()), writers*perWriter; got != want {
			t.Fatalf("concurrent append recorded %d diagnostics, want %d", got, want)
		}
		if got, want := s.Count(diag.Error), writers*perWriter; got != want {
			t.Fatalf("Count(Error) = %d, want %d", got, want)
		}
	})
}

func TestSink_Errorf(t *testing.T) {
	t.Parallel()

	t.Run("appends an Error diagnostic with formatted message", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Errorf(position.At("a.go", 1, 1), "boom %d", 1)
		want := diag.Diag{Severity: diag.Error, Pos: position.At("a.go", 1, 1), Message: "boom 1"}
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("Errorf recorded %+v, want %+v", got, want)
		}
	})
}

func TestSink_Warnf(t *testing.T) {
	t.Parallel()

	t.Run("appends a Warn diagnostic with formatted message", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Warnf(position.At("a.go", 2, 1), "warn %s", "x")
		want := diag.Diag{Severity: diag.Warn, Pos: position.At("a.go", 2, 1), Message: "warn x"}
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("Warnf recorded %+v, want %+v", got, want)
		}
	})
}

func TestSink_Infof(t *testing.T) {
	t.Parallel()

	t.Run("appends an Info diagnostic with formatted message", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Infof(position.At("a.go", 3, 1), "info %v", true)
		want := diag.Diag{Severity: diag.Info, Pos: position.At("a.go", 3, 1), Message: "info true"}
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("Infof recorded %+v, want %+v", got, want)
		}
	})
}

func TestSink_Internalf(t *testing.T) {
	t.Parallel()

	t.Run("appends an Internal diagnostic with formatted message", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Internalf(position.Pos{}, "internal %s", "fault")
		want := diag.Diag{Severity: diag.Internal, Message: "internal fault"}
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("Internalf recorded %+v, want %+v", got, want)
		}
	})
}

func TestSink_Count(t *testing.T) {
	t.Parallel()

	build := func() *diag.Sink {
		s := diag.New()
		s.Errorf(position.Pos{}, "e1")
		s.Errorf(position.Pos{}, "e2")
		s.Warnf(position.Pos{}, "w1")
		s.Infof(position.Pos{}, "i1")
		s.Internalf(position.Pos{}, "int1")
		return s
	}

	cases := []struct {
		name string
		sev  diag.Severity
		want int
	}{
		{"counts Error", diag.Error, 2},
		{"counts Warn", diag.Warn, 1},
		{"counts Info", diag.Info, 1},
		{"counts Internal", diag.Internal, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := build()
			if got := s.Count(tc.sev); got != tc.want {
				t.Fatalf("Count(%v) = %d, want %d", tc.sev, got, tc.want)
			}
		})
	}
}

func TestSink_HasErrors(t *testing.T) {
	t.Parallel()

	t.Run("empty sink reports no errors", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		if s.HasErrors() {
			t.Fatalf("empty sink reported errors")
		}
	})

	t.Run("warnings and infos do not count as errors", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Warnf(position.Pos{}, "warn")
		s.Infof(position.Pos{}, "info")
		if s.HasErrors() {
			t.Fatalf("warnings and infos should not register as errors")
		}
	})

	t.Run("Error severity reports errors", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Errorf(position.Pos{}, "boom")
		if !s.HasErrors() {
			t.Fatalf("Error severity should register")
		}
	})

	t.Run("Internal severity reports errors", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Internalf(position.Pos{}, "panic")
		if !s.HasErrors() {
			t.Fatalf("Internal severity should register as an error")
		}
	})
}

func TestSink_Diagnostics(t *testing.T) {
	t.Parallel()

	t.Run("returns a copy that does not alias internal storage", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Errorf(position.Pos{}, "first")
		snapshot := s.Diagnostics()
		snapshot[0].Message = "MUTATED"
		if s.Diagnostics()[0].Message == "MUTATED" {
			t.Fatalf("Diagnostics() must return a copy")
		}
	})

	t.Run("preserves insertion order", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.Infof(position.Pos{}, "first")
		s.Warnf(position.Pos{}, "second")
		s.Errorf(position.Pos{}, "third")
		got := s.Diagnostics()
		switch {
		case len(got) != 3,
			got[0].Message != "first",
			got[1].Message != "second",
			got[2].Message != "third":
			t.Fatalf("insertion order lost: %+v", got)
		}
	})
}

func TestSink_For(t *testing.T) {
	t.Parallel()

	t.Run("returns a PluginSink that stamps the supplied name", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		p := s.For("repogen")
		if p == nil {
			t.Fatalf("For returned nil")
		}
		if got := p.Plugin(); got != "repogen" {
			t.Fatalf("Plugin = %q, want %q", got, "repogen")
		}
	})
}
