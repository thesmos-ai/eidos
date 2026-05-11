// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/sink"
)

// recordingSink wraps the user-supplied [sink.Sink] and captures
// every Write into an internal map keyed by [emit.Target]. At run
// end the pipeline reads the captured set, hashes each payload, and
// writes the resulting [manifest.Manifest] to disk.
//
// The wrapper preserves write semantics — the inner sink still
// receives every payload — and is concurrent-safe so parallel
// backend file rendering can dispatch without coordination.
type recordingSink struct {
	mu    sync.Mutex
	inner sink.Sink
	files map[emit.Target][]byte
}

// newRecordingSink returns a recordingSink wrapping inner.
func newRecordingSink(inner sink.Sink) *recordingSink {
	return &recordingSink{inner: inner, files: map[emit.Target][]byte{}}
}

// Write delegates to the inner sink and, on success, captures a
// copy of the payload keyed by target for later manifest assembly.
// Write errors from the inner sink propagate verbatim; no capture
// happens on failure.
func (r *recordingSink) Write(target emit.Target, body []byte) error {
	if err := r.inner.Write(target, body); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	dup := make([]byte, len(body))
	copy(dup, body)
	r.files[target] = dup
	return nil
}

// asManifest produces a [manifest.Manifest] from every captured
// write. Each entry's Hash is the SHA-256 hex digest of the
// payload prefixed with "sha256:"; Plugins is left empty because
// the pipeline does not yet have per-plugin write attribution (the
// backend writes through ctx.Sink without identifying which plugin
// contributed which payload).
func (r *recordingSink) asManifest(runID string) *manifest.Manifest {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := manifest.New(runID)
	for target, body := range r.files {
		sum := sha256.Sum256(body)
		m.Add(manifest.Output{
			Target: target,
			Hash:   "sha256:" + hex.EncodeToString(sum[:]),
		})
	}
	return m
}
