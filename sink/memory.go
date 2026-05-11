// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink

import (
	"fmt"
	"maps"
	"sync"

	"go.thesmos.sh/eidos/emit"
)

// Memory is an in-memory [Sink] keyed by [emit.Target]. It is the
// default sink for tests and for the in-process pipeline embedding —
// no filesystem side effects, the captured bytes are inspectable
// directly after Render. Concurrent-safe via mutex.
//
// Successive writes to the same Target overwrite — the contract
// matches the disk sink (which replaces files atomically), so test
// expectations match production semantics.
type Memory struct {
	mu    sync.Mutex
	files map[emit.Target][]byte
}

// NewMemory returns an empty Memory sink ready for use.
func NewMemory() *Memory {
	return &Memory{files: map[emit.Target][]byte{}}
}

// Write records body under target. Returns [ErrInvalidTarget]
// wrapped with the offending target when target.Filename is empty.
func (m *Memory) Write(target emit.Target, body []byte) error {
	if target.Filename == "" {
		return fmt.Errorf("%w: %+v", ErrInvalidTarget, target)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := make([]byte, len(body))
	copy(stored, body)
	m.files[target] = stored
	return nil
}

// Get returns the bytes recorded under target along with true; when
// no entry matches, it returns nil and false. The returned slice is
// a copy callers may mutate without affecting the sink's state.
func (m *Memory) Get(target emit.Target) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, ok := m.files[target]
	if !ok {
		return nil, false
	}
	out := make([]byte, len(body))
	copy(out, body)
	return out, true
}

// Files returns a snapshot of every recorded entry. The returned
// map and byte slices are independent copies — mutating them does
// not affect the sink.
func (m *Memory) Files() map[emit.Target][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[emit.Target][]byte, len(m.files))
	for k, v := range m.files {
		dup := make([]byte, len(v))
		copy(dup, v)
		out[k] = dup
	}
	return out
}

// Len returns the number of recorded entries.
func (m *Memory) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.files)
}

// Clear removes every recorded entry. Useful between test cases
// against a shared sink instance.
func (m *Memory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	maps.DeleteFunc(m.files, func(emit.Target, []byte) bool { return true })
}
