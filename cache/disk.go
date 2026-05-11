// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Disk is a filesystem-backed [Cache] rooted at a configurable
// directory. Entries are stored under "<root>/<key-prefix>/<key>"
// with a deterministic ".eidos.tmp" suffix used during writes for
// atomic temp+rename. The prefix split (first two hex characters of
// the key) keeps the per-directory entry count manageable for keys
// that share a common hash prefix.
//
// Concurrent writers to the same key are serialised through a
// per-instance mutex; entries are content-addressed so concurrent
// Puts converge on the same value regardless of the order they hit
// the disk.
type Disk struct {
	root string
	mu   sync.Mutex
}

// tempSuffix is appended to a key path while a Put is in progress.
// The same value is reused across writes; the per-instance mutex
// guarantees only one write per key is in flight at a time.
const tempSuffix = ".eidos.tmp"

// NewDisk returns a Disk cache rooted at root. The directory is
// created lazily on the first [Disk.Put]; callers do not need to
// ensure it exists.
func NewDisk(root string) *Disk {
	return &Disk{root: root}
}

// Root returns the configured root directory.
func (d *Disk) Root() string { return d.root }

// Get returns the bytes cached under key. Missing entries return
// (nil, false); read errors other than "not found" return (nil,
// false) as well — the caller treats a missing cache entry the same
// as one that failed to read, falling back to a fresh recompute.
//
// An empty key is a programmer error and returns (nil, false) — the
// non-error miss matches the spirit of the interface, where Get is
// not allowed to surface non-existence as an error.
func (d *Disk) Get(key string) ([]byte, bool) {
	if key == "" {
		return nil, false
	}
	body, err := os.ReadFile(d.keyPath(key))
	if err != nil {
		return nil, false
	}
	return body, true
}

// Put stores value under key atomically via temp+rename in the
// key's directory. Returns [ErrInvalidKey] when key is empty;
// filesystem errors propagate wrapped with the destination path
// for diagnostics.
func (d *Disk) Put(key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("%w: empty", ErrInvalidKey)
	}
	full := d.keyPath(key)
	dir := filepath.Dir(full)
	tmpPath := full + tempSuffix

	d.mu.Lock()
	defer d.mu.Unlock()

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("cache: mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(tmpPath, value, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("cache: write %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, full); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("cache: rename %s -> %s: %w", tmpPath, full, err)
	}
	return nil
}

// Has reports whether key is present without reading the value.
// Useful for tooling (eidos explain, --verify-cache) that needs to
// enumerate the cache state without paying the read cost.
func (d *Disk) Has(key string) bool {
	if key == "" {
		return false
	}
	_, err := os.Stat(d.keyPath(key))
	return err == nil
}

// keyPath returns the filesystem path entries for key are stored
// under. Keys of length >= 2 split into "<root>/<key[:2]>/<key>";
// shorter keys go directly under root. Empty keys are rejected by
// the caller before reaching this function.
func (d *Disk) keyPath(key string) string {
	if len(key) >= 2 {
		return filepath.Join(d.root, key[:2], key)
	}
	return filepath.Join(d.root, key)
}
