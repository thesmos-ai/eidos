// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink

import (
	"errors"

	"go.thesmos.sh/eidos/emit"
)

// Multi is a [Sink] that fans every [Sink.Write] call out to a list
// of underlying sinks in registration order. Errors from individual
// sinks are joined via [errors.Join] and returned together so the
// caller sees every failure rather than just the first.
//
// Common composition: a [Disk] sink for the persistent output plus a
// [Memory] sink for in-process inspection (assertions, tooling).
type Multi struct {
	sinks []Sink
}

// NewMulti returns a Multi that dispatches to sinks in the order
// supplied. Nil sinks are filtered out.
func NewMulti(sinks ...Sink) *Multi {
	out := make([]Sink, 0, len(sinks))
	for _, s := range sinks {
		if s != nil {
			out = append(out, s)
		}
	}
	return &Multi{sinks: out}
}

// Sinks returns a copy of the underlying sink list. Useful for
// tests that want to verify composition without reaching into the
// struct.
func (m *Multi) Sinks() []Sink {
	out := make([]Sink, len(m.sinks))
	copy(out, m.sinks)
	return out
}

// Write dispatches to every underlying sink in registration order.
// All sinks are invoked even if earlier ones fail; the joined error
// surfaces every failure to the caller.
func (m *Multi) Write(target emit.Target, body []byte) error {
	var errs []error
	for _, s := range m.sinks {
		if err := s.Write(target, body); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
