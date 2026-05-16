// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package inprocesstest_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/acceptancetest"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	frontendgolang "go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/plugins/generator/mock"
	"go.thesmos.sh/eidos/plugins/generator/mocktest"
	"go.thesmos.sh/eidos/sink"
)

// TestIntegration_RenderedOutputCompiles pins byte-level
// correctness for the mock + mocktest pair end-to-end across both
// routing modes the framework supports: the default external-test
// package shift on `_test.go` files, and the testkit sibling-
// package mode driven by `+gen:out`.
//
// For each fixture the test runs the pipeline against a self-
// contained module copied into a per-case tempdir, asserts every
// expected generated path exists, and runs `go vet ./...` over
// the resulting tree. Vet compiles test files too — the only level
// at which renderer surprises in the `Test<MockName>` function
// (named-return semantics, refs into the mock package across the
// `_test`-shift boundary, anonymous-param func literals) surface
// as real errors.
func TestIntegration_RenderedOutputCompiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		fixture  string
		expected []string
	}{
		{
			// Default routing: mock lands alongside source in the
			// source package; mocktest lands alongside source in the
			// external-test package (`_test.go` shift). External refs
			// into the mock package qualify with `store.` since the
			// test file is in `store_test`.
			name:    "default routing places test in external _test pkg",
			fixture: "testdata/integrationfx",
			expected: []string{
				"store/store_mock.go",
				"store/store_mock_test.go",
			},
		},
		{
			// Testkit pattern via standalone +gen:out: routes both
			// generated files into a sibling `storetest/` package.
			// mocktest's `_test.go` shift fires on top, landing the
			// test in `package storetest_test` while the mock itself
			// is in `package storetest`. Refs into the mock package
			// pick up the `storetest.` qualifier in the test file.
			name:    "+gen:out sibling dir routes both files into sibling pkg",
			fixture: "testdata/testkitfx",
			expected: []string{
				"store/storetest/store_mock.go",
				"store/storetest/store_mock_test.go",
			},
		},
		{
			// Per-directive routing: `+gen:mock out=... pkg=...`
			// bundles the routing override into the generator
			// directive that triggers emission. The framework
			// auto-recognises `out=` and `pkg=` on any owner
			// directive, scoping the override to the owning plugin.
			// Semantically identical to the standalone `+gen:out`
			// form above, but read at the directive that actually
			// produces the emission.
			name:    "+gen:mock out= bundles routing into the generator directive",
			fixture: "testdata/perdirectivefx",
			expected: []string{
				"store/storetest/store_mock.go",
				"store/storetest/store_mock_test.go",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			workdir := t.TempDir()
			acceptancetest.CopyDir(t, tc.fixture, workdir)

			pipe := pipelinetest.New(t).
				WithFrontend(frontendgolang.New()).
				WithGenerator(mock.New()).
				WithGenerator(mocktest.New()).
				WithBackend(backendgolang.New()).
				WithSink(sink.NewDisk(workdir)).
				WithPluginOptions(frontendgolang.FrontendName, map[string]string{
					"dir":              workdir,
					"ignore_workspace": "true",
				}).
				Build().
				Run("./...")

			for _, d := range pipe.Diagnostics().Diagnostics() {
				if d.Severity == diag.Error || d.Severity == diag.Internal {
					t.Fatalf("unexpected %s diagnostic from %s: %s\n%s",
						d.Severity, d.Plugin, d.Message, d.Detail)
				}
			}

			for _, rel := range tc.expected {
				if _, err := os.Stat(filepath.Join(workdir, rel)); err != nil {
					t.Errorf("expected generated file %s missing: %v", rel, err)
				}
			}
			if t.Failed() {
				dumpWorkdir(t, workdir, pipe.Diagnostics())
			}

			cmd := exec.CommandContext(t.Context(), "go", "vet", "./...")
			cmd.Dir = workdir
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("go vet of rendered output failed: %v\nstderr:\n%s",
					err, stderr.String())
			}
		})
	}
}

// dumpWorkdir prints every file under workdir plus every recorded
// diagnostic to t.Logf. Called only on test failure so the
// rendered output and pipeline state are visible in the failure
// transcript without polluting passing runs.
func dumpWorkdir(t *testing.T, workdir string, d *diag.Sink) {
	t.Helper()
	_ = filepath.Walk(workdir, func(p string, info os.FileInfo, err error) error {
		switch {
		case err != nil:
			return err
		case info.IsDir():
			return nil
		}
		t.Logf("tree: %s", p)
		if data, rerr := os.ReadFile(p); rerr == nil { //nolint:gosec // controlled testdata path
			t.Logf("--- %s ---\n%s", p, string(data))
		}
		return nil
	})
	for _, diagEntry := range d.Diagnostics() {
		t.Logf("diag[%s]: %s — %s",
			diagEntry.Severity, diagEntry.Plugin, diagEntry.Message)
	}
}
