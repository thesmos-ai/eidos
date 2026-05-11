// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package testpipe

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// updateGolden is the registered `-update-golden` flag. When set,
// [FileAssertion.MatchesGolden] rewrites the golden file at the
// supplied path with the current file's content instead of asserting
// against it.
//
// The flag is registered at package init via [flag.Bool] against
// [flag.CommandLine]; Go's test runner parses it automatically. Run
// the failing test binary with `-update-golden` to bring the
// fixtures in sync with intended output, then commit the diff.
var updateGolden = flag.Bool("update-golden", false,
	"rewrite testpipe golden files from the current run's output")

// UpdateGolden reports whether the `-update-golden` flag is set on
// the current test invocation. Tests that build their own golden
// comparison helpers (e.g. directory-level diffs) consult this to
// decide between assert and rewrite modes.
func UpdateGolden() bool { return *updateGolden }

// MatchesGolden compares the file's content against the bytes stored
// at path. When the `-update-golden` flag is set, MatchesGolden
// rewrites path atomically with the current bytes (temp + rename) so
// a partially-written golden never ends up on disk. The directory
// containing path is created if it does not already exist.
//
// A missing golden file is treated as a test failure in assert mode
// and as a fresh write in update mode. Read failures other than
// "missing file" always fail the test — they typically indicate a
// permissions problem worth fixing rather than a stale golden.
func (a *FileAssertion) MatchesGolden(path string) *FileAssertion {
	a.t.Helper()
	if *updateGolden {
		if err := writeGolden(path, a.body); err != nil {
			a.t.Fatalf("testpipe: failed to rewrite golden %s: %v", path, err)
		}
		return a
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			a.t.Errorf("testpipe: golden file %s missing; run with -update-golden to create it", path)
			return a
		}
		a.t.Fatalf("testpipe: failed to read golden %s: %v", path, err)
	}
	if !bytes.Equal(a.body, expected) {
		a.t.Errorf(
			"testpipe: file %s does not match golden %s\n----- got -----\n%s\n----- want -----\n%s\n----- end -----",
			a.target.JoinPath(), path, a.body, expected,
		)
	}
	return a
}

// writeGolden atomically writes body to path. The temp file lives in
// the same directory as path so the rename is a same-filesystem
// operation and therefore atomic on every supported OS. Parent
// directories are created with 0o755 when missing.
func writeGolden(path string, body []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return fmt.Errorf("write temp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}
