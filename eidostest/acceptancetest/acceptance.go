// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package acceptancetest drives the [cmd/eidos-reference]
// binary as a black box for end-to-end scenario testing — the
// layer above [pipelinetest] / [backendtest] / [frontendtest]
// that exercises everything from process startup through
// rendered files on disk.
//
// The harness builds the binary once per process via
// [BuildBinary] (cached behind a [sync.Once] so multiple tests
// share the build) and runs it with [RunCmd]. Tests that need
// a working directory invoke the binary against a tempdir
// populated from a testdata fixture.
//
// The acceptance layer catches behaviour the library-level
// harnesses cannot: exit-code propagation, signal handling,
// stdout / stderr buffering and interleaving, flag parsing,
// config-file discovery, manifest persistence, and any
// cross-cutting concerns that only manifest when an actual
// process runs. Use it for high-confidence regression tests of
// the binary's user-facing surface; use [pipelinetest] for
// faster in-process scenarios.
package acceptancetest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// binaryImportPath is the Go module path of the
// reference-binary entry point. The workspace's go.work
// resolves it to the local cmd/eidos-reference directory; a
// downstream consumer running these tests without the
// workspace would need the module published or replaced.
const binaryImportPath = "go.thesmos.sh/eidos/cmd/eidos-reference"

//nolint:gochecknoglobals // process-wide cache of the built binary path
var (
	binaryOnce sync.Once
	binaryPath string
	errBinary  error
)

// BuildBinary builds the reference binary and returns the path
// to the resulting executable. The build is cached per process
// via [sync.Once] so multiple test functions share one binary.
//
// Callers do not delete the resulting file — the [os.MkdirTemp]
// directory is left in place for the process lifetime and the
// OS reclaims it on next reboot. Tests using `go test` see one
// build per `go test` invocation.
//
// A build failure fails the test via [t.Fatalf] with the
// underlying compiler error attached.
func BuildBinary(t *testing.T) string {
	t.Helper()
	binaryOnce.Do(func() {
		// MkdirTemp (not t.TempDir) because the sync.Once
		// caches across every test in the process; t.TempDir
		// would be reaped when the first caller's test ends,
		// invalidating the path for every subsequent test.
		dir, err := os.MkdirTemp("", "eidos-acceptance-*") //nolint:usetesting
		if err != nil {
			errBinary = err
			return
		}
		path := filepath.Join(dir, "eidos-reference")
		cmd := exec.CommandContext(t.Context(), "go", "build", "-o", path, binaryImportPath)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			errBinary = err
			return
		}
		binaryPath = path
	})
	if errBinary != nil {
		t.Fatalf("acceptancetest: build %s: %v", binaryImportPath, errBinary)
	}
	return binaryPath
}

// Result captures one [RunCmd] invocation's outputs. Stdout and
// Stderr are the binary's captured byte streams; ExitCode is the
// raw process exit code, including non-zero values from
// [cli.ExitUserError] / [cli.ExitPipelineError] / …
type Result struct {
	// Stdout is the binary's stdout stream as a single byte
	// string. Tests substring-match or unmarshal it for
	// command-specific assertions.
	Stdout string

	// Stderr is the binary's stderr stream as a single byte
	// string. Diagnostics in text format land here; tests
	// inspect it for expected error messages.
	Stderr string

	// ExitCode is the raw process exit code: 0 for success,
	// non-zero for any failure family the binary reports. See
	// [cli.ExitOK] / [cli.ExitUserError] / [cli.ExitPipelineError]
	// for the canonical mapping.
	ExitCode int
}

// CopyDir recursively copies the contents of src into dst.
// Used by fixture-driven acceptance tests: copy a testdata
// project tree into a per-test [t.TempDir] and run the binary
// against the copy so the original testdata stays untouched.
//
// dst must already exist (typically `t.TempDir()`). Subdirectory
// structure under src is recreated; symlinks and special files
// are not supported (they don't appear in the testdata trees
// the framework ships).
func CopyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return fmt.Errorf("rel %s: %w", path, relErr)
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
			return nil
		}
		// gosec G122 / G703: this harness walks controlled
		// testdata under the project tree; symlink-TOCTOU is
		// not in the threat model for test fixtures.
		data, readErr := os.ReadFile(path) //nolint:gosec
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil { //nolint:gosec
			return fmt.Errorf("write %s: %w", target, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("acceptancetest: CopyDir %s → %s: %v", src, dst, err)
	}
}

// RunCmd invokes the reference binary with args from workdir
// and captures stdout / stderr / exit-code into a [Result].
// Workdir may be empty; the binary runs in its parent process's
// cwd in that case.
//
// A child-process failure to start (binary missing, fork error,
// …) fails the test via [t.Fatalf]; a non-zero process exit is
// not an error from the harness's perspective — tests assert on
// [Result.ExitCode] directly. This lets the same helper drive
// both success and failure paths without special-casing.
func RunCmd(t *testing.T, workdir string, args ...string) Result {
	t.Helper()
	bin := BuildBinary(t)
	cmd := exec.CommandContext(t.Context(), bin, args...)
	if workdir != "" {
		cmd.Dir = workdir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("acceptancetest: invoke binary: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}
	return Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}
