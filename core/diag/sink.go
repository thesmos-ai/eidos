// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package diag

import (
	"fmt"
	"slices"
	"sync"

	"go.thesmos.sh/eidos/core/position"
)

// Sink is the central diagnostic collector for one pipeline run.
//
// All methods are safe to call concurrently; the underlying mutex is
// uncontended in the common case (one plugin emits at a time) and
// short-held when contended. Diagnostics are appended in the order
// they are emitted; rendering preserves that order.
//
// The zero value is unusable; construct via [New].
type Sink struct {
	mu      sync.Mutex
	diags   []Diag
	discard bool
}

// New returns a freshly-initialised Sink with no diagnostics.
func New() *Sink {
	return &Sink{}
}

// Append records d as-is without validation. Callers that want
// formatting should use [Sink.Errorf] / [Sink.Warnf] / [Sink.Infof] or
// emit through a [PluginSink] obtained from [Sink.For]. Discard
// sinks (constructed via [Discard]) silently drop d.
func (s *Sink) Append(d Diag) {
	if s.discard {
		return
	}
	s.mu.Lock()
	s.diags = append(s.diags, d)
	s.mu.Unlock()
}

// Errorf appends an Error diagnostic with no plugin attribution.
// Callers inside a plugin should obtain a [PluginSink] via [Sink.For]
// and use its method instead.
func (s *Sink) Errorf(pos position.Pos, format string, args ...any) {
	s.Append(Diag{Severity: Error, Pos: pos, Message: fmt.Sprintf(format, args...)})
}

// Warnf appends a Warn diagnostic with no plugin attribution.
func (s *Sink) Warnf(pos position.Pos, format string, args ...any) {
	s.Append(Diag{Severity: Warn, Pos: pos, Message: fmt.Sprintf(format, args...)})
}

// Infof appends an Info diagnostic with no plugin attribution.
func (s *Sink) Infof(pos position.Pos, format string, args ...any) {
	s.Append(Diag{Severity: Info, Pos: pos, Message: fmt.Sprintf(format, args...)})
}

// Internalf appends an Internal diagnostic with no plugin attribution.
// Reserved for core / framework bugs surfaced from outside any plugin.
func (s *Sink) Internalf(pos position.Pos, format string, args ...any) {
	s.Append(Diag{Severity: Internal, Pos: pos, Message: fmt.Sprintf(format, args...)})
}

// For returns a [PluginSink] that stamps the supplied plugin name on
// every diagnostic it emits. This is the canonical way for plugin
// authors to obtain a diagnostic interface — they pass the result of
// `ctx.Diag.For(p.Name())` to helpers within the plugin.
func (s *Sink) For(plugin string) *PluginSink {
	return &PluginSink{sink: s, plugin: plugin}
}

// Diagnostics returns a snapshot copy of every diagnostic recorded so
// far, in insertion order. The result does not alias the Sink's
// storage — callers may sort or filter without affecting later
// emission.
func (s *Sink) Diagnostics() []Diag {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.diags)
}

// Count returns how many diagnostics of the given severity have been
// recorded.
func (s *Sink) Count(sev Severity) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, d := range s.diags {
		if d.Severity == sev {
			n++
		}
	}
	return n
}

// HasErrors reports whether any Error or Internal diagnostic has been
// recorded. Used by the pipeline runner to decide the final exit
// disposition after every phase has run to completion.
func (s *Sink) HasErrors() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.diags {
		if d.Severity >= Error {
			return true
		}
	}
	return false
}
