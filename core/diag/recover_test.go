// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
)

// triggerPanic is a tiny helper used solely to provoke a recover-able
// panic from a function the test fully controls.
func triggerPanic(value any) { panic(value) }

func TestRecoverAs(t *testing.T) {
	t.Parallel()

	t.Run("no panic is a no-op", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		func() {
			defer diag.RecoverAs(s.For("p"), position.Pos{})
		}()
		if got := s.Diagnostics(); len(got) != 0 {
			t.Fatalf("RecoverAs without a panic should not record anything; got %+v", got)
		}
	})

	t.Run("records error diagnostic attributed to the plugin", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		pos := position.At("a.go", 42, 1)
		func() {
			defer diag.RecoverAs(s.For("shape-writer"), pos)
			triggerPanic("kaboom")
		}()

		diags := s.Diagnostics()
		if len(diags) != 1 {
			t.Fatalf("expected one diagnostic; got %d (%+v)", len(diags), diags)
		}
		d := diags[0]
		if d.Severity != diag.Error {
			t.Fatalf("severity = %v, want Error", d.Severity)
		}
		if d.Plugin != "shape-writer" {
			t.Fatalf("plugin = %q, want %q", d.Plugin, "shape-writer")
		}
		if d.Pos != pos {
			t.Fatalf("pos = %+v, want %+v", d.Pos, pos)
		}
		if !strings.Contains(d.Message, "shape-writer") || !strings.Contains(d.Message, "kaboom") {
			t.Fatalf("message %q should mention plugin name and panic value", d.Message)
		}
		if d.Detail == "" {
			t.Fatalf("detail should contain the recovered stack trace")
		}
	})
}

func TestRecoverInternal(t *testing.T) {
	t.Parallel()

	t.Run("no panic is a no-op", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		func() {
			defer diag.RecoverInternal(s, position.Pos{}, "core/pipeline.run")
		}()
		if got := s.Diagnostics(); len(got) != 0 {
			t.Fatalf("RecoverInternal without a panic should not record anything; got %+v", got)
		}
	})

	t.Run("records internal diagnostic with no plugin attribution", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		func() {
			defer diag.RecoverInternal(s, position.Pos{}, "core/pipeline.run")
			triggerPanic("framework bug")
		}()

		diags := s.Diagnostics()
		if len(diags) != 1 {
			t.Fatalf("expected one diagnostic; got %d (%+v)", len(diags), diags)
		}
		d := diags[0]
		if d.Severity != diag.Internal {
			t.Fatalf("severity = %v, want Internal", d.Severity)
		}
		if d.Plugin != "" {
			t.Fatalf("plugin = %q, internal diagnostics carry no plugin attribution", d.Plugin)
		}
		if !strings.Contains(d.Message, "core/pipeline.run") {
			t.Fatalf("message should name the panic location; got %q", d.Message)
		}
		if !strings.Contains(d.Message, "framework bug") {
			t.Fatalf("message should include the panic value; got %q", d.Message)
		}
		if d.Detail == "" {
			t.Fatalf("detail should contain the recovered stack trace")
		}
	})
}
