// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
)

func TestPluginSink_Plugin(t *testing.T) {
	t.Parallel()

	t.Run("returns the plugin name supplied to Sink.For", func(t *testing.T) {
		t.Parallel()
		p := diag.New().For("repogen")
		if got := p.Plugin(); got != "repogen" {
			t.Fatalf("Plugin = %q, want %q", got, "repogen")
		}
	})
}

func TestPluginSink_Errorf(t *testing.T) {
	t.Parallel()

	t.Run("appends an Error diagnostic attributed to the plugin", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.For("p").Errorf(position.At("a.go", 1, 1), "e %d", 1)
		want := diag.Diag{Severity: diag.Error, Plugin: "p", Pos: position.At("a.go", 1, 1), Message: "e 1"}
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("Errorf recorded %+v, want %+v", got, want)
		}
	})
}

func TestPluginSink_Warnf(t *testing.T) {
	t.Parallel()

	t.Run("appends a Warn diagnostic attributed to the plugin", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.For("p").Warnf(position.At("a.go", 2, 1), "w %s", "x")
		want := diag.Diag{Severity: diag.Warn, Plugin: "p", Pos: position.At("a.go", 2, 1), Message: "w x"}
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("Warnf recorded %+v, want %+v", got, want)
		}
	})
}

func TestPluginSink_Infof(t *testing.T) {
	t.Parallel()

	t.Run("appends an Info diagnostic attributed to the plugin", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.For("p").Infof(position.At("a.go", 3, 1), "i %v", false)
		want := diag.Diag{Severity: diag.Info, Plugin: "p", Pos: position.At("a.go", 3, 1), Message: "i false"}
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("Infof recorded %+v, want %+v", got, want)
		}
	})
}

func TestPluginSink_AppendDetail(t *testing.T) {
	t.Parallel()

	t.Run("records a diagnostic carrying both message and multi-line detail", func(t *testing.T) {
		t.Parallel()
		s := diag.New()
		s.For("repogen").AppendDetail(
			diag.Error,
			position.At("a.go", 1, 1),
			"msg",
			"stack trace line\n  another line",
		)
		want := diag.Diag{
			Severity: diag.Error,
			Plugin:   "repogen",
			Pos:      position.At("a.go", 1, 1),
			Message:  "msg",
			Detail:   "stack trace line\n  another line",
		}
		got := s.Diagnostics()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("AppendDetail recorded %+v, want %+v", got, want)
		}
	})
}
