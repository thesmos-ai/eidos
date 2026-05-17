// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package acceptancetest_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/eidostest/acceptancetest"
)

// demoFixture is the path to the in-tree demoproject testdata
// relative to this package's test files. demoproject ships
// representative +gen: directives covering every reference
// plugin (repo / builder / mock / register) and is the
// canonical end-to-end acceptance fixture.
const demoFixture = "../testdata/demoproject"

// TestRunOnDemoProject pins the binary against the demoproject
// fixture as a full end-to-end scenario: run produces the
// expected generated files alongside source, exits cleanly,
// and writes a manifest. This is the high-value acceptance
// test — every plugin in the pipeline contributes, the routing
// layer composes filenames, and the backend renders + writes
// real Go files.
func TestRunOnDemoProject(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	acceptancetest.CopyDir(t, demoFixture, workdir)

	res := acceptancetest.RunCmd(t, workdir, "run", "./...")
	if res.ExitCode != 0 {
		t.Fatalf("run on demoproject exit %d\nstderr:\n%s\nstdout:\n%s",
			res.ExitCode, res.Stderr, res.Stdout)
	}

	// Generated files appear alongside source — one per
	// (plugin, source-struct) pair that opted in via +gen: directive.
	for _, rel := range []string{
		// repogen targets blog.Article and blog.User
		"blog/article_repo.go",
		"blog/user_repo.go",

		// buildergen targets blog.Article, blog.User, blog.Comment
		"blog/article_builder.go",
		"blog/user_builder.go",
		"blog/comment_builder.go",

		// registrygen targets blog.Article
		"blog/article_registry.go",

		// mockgen targets the user-authored blog.Searcher interface
		// plus the repogen-emitted Article/User Repository interfaces
		"blog/searcher_mock_test.go",
		"blog/article_mock_test.go",
		"blog/user_mock_test.go",

		// enum (multi-output) targets blog.Status — production
		// surface in _enum.go, paired round-trip tests in
		// _enum_test.go (auto-shifted to package blog_test).
		"blog/status_enum.go",
		"blog/status_enum_test.go",
	} {
		path := filepath.Join(workdir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected generated file %s missing: %v", rel, err)
		}
	}

	// The manifest records the run for change tracking + prune
	// + check workflows.
	manifestPath := filepath.Join(workdir, ".eidos-reference", "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest.json should be written; got: %v", err)
	}
}

// TestRunOnDemoProject_GeneratedOutputCompiles pins the
// byte-level correctness of the generated output: after the
// binary runs, `go build ./...` of the demoproject tree must
// succeed. A regression in any backend rendering pass, import
// resolution, or `gofmt` finalisation surfaces here as a
// compile error.
func TestRunOnDemoProject_GeneratedOutputCompiles(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	acceptancetest.CopyDir(t, demoFixture, workdir)

	runRes := acceptancetest.RunCmd(t, workdir, "run", "./...")
	if runRes.ExitCode != 0 {
		t.Fatalf("run exit %d\nstderr:\n%s", runRes.ExitCode, runRes.Stderr)
	}

	cmd := exec.CommandContext(t.Context(), "go", "build", "./...")
	cmd.Dir = workdir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build of generated demoproject failed: %v\nstderr:\n%s",
			err, stderr.String())
	}
}

// TestRunOnDemoProject_IsIdempotent pins the determinism
// contract end-to-end: running the binary twice against the
// same source tree produces byte-identical generated files.
// Non-deterministic output anywhere in the pipeline (map
// iteration, time-derived values, unstable import sets,
// non-deterministic slot ordering) surfaces as a snapshot diff
// across the two runs.
func TestRunOnDemoProject_IsIdempotent(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	acceptancetest.CopyDir(t, demoFixture, workdir)

	firstRun := acceptancetest.RunCmd(t, workdir, "run", "./...")
	if firstRun.ExitCode != 0 {
		t.Fatalf("first run exit %d\nstderr:\n%s", firstRun.ExitCode, firstRun.Stderr)
	}
	first := snapshotGenerated(t, workdir)

	secondRun := acceptancetest.RunCmd(t, workdir, "run", "./...")
	if secondRun.ExitCode != 0 {
		t.Fatalf("second run exit %d\nstderr:\n%s", secondRun.ExitCode, secondRun.Stderr)
	}
	second := snapshotGenerated(t, workdir)

	if !maps.Equal(first, second) {
		t.Errorf("generated tree differs across two runs (non-deterministic output)")
		for path, hash1 := range first {
			if hash2, ok := second[path]; !ok {
				t.Errorf("  file disappeared on second run: %s", path)
			} else if hash1 != hash2 {
				t.Errorf("  file content changed: %s (%s → %s)", path, hash1[:12], hash2[:12])
			}
		}
		for path := range second {
			if _, ok := first[path]; !ok {
				t.Errorf("  file appeared only on second run: %s", path)
			}
		}
	}
}

// snapshotGenerated returns a path→content-hash map of every
// `*_repo.go`, `*_builder.go`, `*_registry.go`, `*_mock_test.go`
// file under workdir. Used by the idempotency check to compare
// two pipeline runs without holding their byte contents in
// memory.
func snapshotGenerated(t *testing.T, workdir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(workdir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		generated := strings.HasSuffix(name, "_repo.go") ||
			strings.HasSuffix(name, "_builder.go") ||
			strings.HasSuffix(name, "_registry.go") ||
			strings.HasSuffix(name, "_mock_test.go")
		if !generated {
			return nil
		}
		// gosec G122: controlled test workdir, no symlink threat.
		data, readErr := os.ReadFile(path) //nolint:gosec
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		rel, relErr := filepath.Rel(workdir, path)
		if relErr != nil {
			return fmt.Errorf("rel %s: %w", path, relErr)
		}
		sum := sha256.Sum256(data)
		out[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotGenerated: %v", err)
	}
	return out
}
