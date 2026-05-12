// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/writer"
)

// Disk is a [Sink] that writes to the filesystem under a configured
// root directory. Each [Sink.Write] writes the body to a sibling
// temporary file (suffix ".eidos.tmp") and then renames it over the
// final path. The rename is atomic on POSIX filesystems because the
// temporary file lives in the same directory as the destination, so
// a crash or concurrent reader never observes a partial file.
//
// Disk creates any missing intermediate directories under the root
// at write time. The root itself must exist or be creatable; the
// sink does not validate it at construction.
//
// A package-level mutex serialises writes so concurrent renders to
// the same target produce a single deterministic outcome and the
// deterministic temp-file suffix never collides with itself.
type Disk struct {
	root string
	mu   sync.Mutex
}

// tempSuffix is the suffix appended to the destination path to form
// the temporary file used by the atomic-rename strategy. Kept in
// the same directory as the destination so [os.Rename] is atomic.
const tempSuffix = ".eidos.tmp"

// NewDisk returns a Disk sink rooted at root. Paths derived from
// [emit.Target.Dir] / [emit.Target.Filename] join under root.
func NewDisk(root string) *Disk {
	return &Disk{root: root}
}

// Root returns the configured root directory.
func (d *Disk) Root() string { return d.root }

// Write writes body to "<root>/<target.Dir>/<target.Filename>"
// atomically. Returns [ErrInvalidTarget] when target.Filename is
// empty. Filesystem errors propagate wrapped with the destination
// path for diagnostics.
//
// An absolute target.Dir bypasses root — used for routes that
// already encode a workdir-anchored path (typically because the
// router resolved the Dir from a source file's absolute path).
// Relative target.Dir joins under root.
func (d *Disk) Write(target emit.Target, body []byte) error {
	if target.Filename == "" {
		return fmt.Errorf("%w: %+v", ErrInvalidTarget, target)
	}
	var full string
	if filepath.IsAbs(target.Dir) {
		full = filepath.Join(target.Dir, target.Filename)
	} else {
		full = filepath.Join(d.root, target.Dir, target.Filename)
	}
	dir := filepath.Dir(full)
	tmpPath := full + tempSuffix

	d.mu.Lock()
	defer d.mu.Unlock()

	// Skip the write when the destination's body matches what we
	// would write. Plain byte-equality catches the simple case;
	// when that fails we compare the provenance trailer the
	// backend stamps over body bytes alone, which lets the
	// short-circuit survive header-only differences (the Command
	// line varies between invocations: `eidos run ./blog/...`
	// versus `eidos run ./...` produce identical bodies but
	// different Command stamps). Idempotency is a load-bearing
	// property: re-running the pipeline against unchanged inputs
	// must not touch the disk regardless of how the user phrased
	// the invocation.
	if existing, err := os.ReadFile(full); err == nil && bodiesMatch(existing, body) {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // generated dirs need group/other-read
		return fmt.Errorf("sink: mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(tmpPath, body, 0o644); err != nil { //nolint:gosec // generated files need group/other-read
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sink: write %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, full); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sink: rename %s -> %s: %w", tmpPath, full, err)
	}
	return nil
}

// bodiesMatch reports whether two rendered file blobs encode the
// same content. The check tries byte-equality first (fast path,
// covers non-generated files and runs with identical headers)
// then falls back to comparing the provenance trailer's stamped
// hash (the brand-stamped digest is over body bytes alone, so
// header-only deltas — Command lines, source-path normalisation
// — don't surface as drift). Files missing the trailer fall
// through to the byte-equal verdict.
func bodiesMatch(a, b []byte) bool {
	if bytes.Equal(a, b) {
		return true
	}
	ah, aOK := writer.ExtractProvenance(a)
	if !aOK {
		return false
	}
	bh, bOK := writer.ExtractProvenance(b)
	if !bOK {
		return false
	}
	return ah == bh
}
