// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Write serialises m as JSON to path atomically: writes to a
// sibling temp file (suffix ".eidos.tmp") and then renames over
// the final path. Creates intermediate directories under path's
// parent at write time.
func Write(path string, m *Manifest) error {
	// Manifest fields are all JSON-marshalable, so MarshalIndent
	// cannot fail; discarding the error keeps the contract explicit
	// and avoids an unreachable defensive branch.
	body, _ := json.MarshalIndent(m, "", "  ") //nolint:errcheck,musttag // Manifest is fully tagged; cannot fail

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("manifest: mkdir %s: %w", dir, err)
	}

	tmpPath := path + ".eidos.tmp"
	if err := os.WriteFile(tmpPath, body, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("manifest: write %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("manifest: rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

// Read parses the manifest at path. Returns the parsed value on
// success; a wrapped filesystem error when the file cannot be
// opened or read; a wrapped JSON error when the file is malformed;
// or [ErrUnsupportedVersion] when [Manifest.Version] does not match
// the [Version] this build understands.
func Read(path string) (*Manifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("manifest: read %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(body, &m); err != nil { //nolint:musttag // Manifest is fully tagged
		return nil, fmt.Errorf("manifest: parse %s: %w", path, err)
	}
	if m.Version != Version {
		return nil, fmt.Errorf("%w: %d (want %d)", ErrUnsupportedVersion, m.Version, Version)
	}
	return &m, nil
}
