// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protopipe_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/bridge/protogo"
	"go.thesmos.sh/eidos/eidostest/protopipe"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/buildergen"
	"go.thesmos.sh/eidos/reference/mockgen"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/sink"
)

// TestToolchain_GoBuildAgainstRenderedOutput drives the proto
// frontend + protogo bridge + buildergen + Go backend pipeline
// against a fixture exercising the bridge's three nontrivial
// composition rules (well-known reference, optional field,
// nested-message reference) and confirms the rendered Go
// compiles under the host toolchain. The test materialises the
// rendered output alongside the fixture's hand-written stubs
// (the directory carries both committed types and the
// pipeline's per-run output), invokes `go build` against the
// directory, and asserts a clean exit. Cleanup removes the
// per-run rendered files so subsequent `go build ./...` against
// the eidos module sees only the committed sources.
//
// The test does not call [testing.T.Parallel] — it writes
// transient files into a committed package directory and must
// not race other tests that touch the same files.
//
//nolint:paralleltest // intentional; mutates a committed package dir.
func TestToolchain_GoBuildAgainstRenderedOutput(t *testing.T) {
	fixtureDir := buildFixtureDir(t)

	mem := sink.NewMemory()
	result := protopipe.Run(t, protopipe.RunOptions{
		SourceDir:  fixtureDir,
		Pattern:    "./...",
		Annotators: []plugin.Annotator{protogo.New()},
		Generators: []plugin.Generator{
			buildergen.New(),
			repogen.New(),
			mockgen.New(),
		},
		Backend: backend_golang.New(),
		Sink:    mem,
	})
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}
	if result.Diag.HasErrors() {
		t.Fatalf("pipeline produced error diagnostics: %+v", result.Diag.Diagnostics())
	}

	rendered := writeRenderedFilesInto(t, mem, fixtureDir)
	if len(rendered) == 0 {
		t.Fatalf("pipeline produced no rendered files for %q", fixtureDir)
	}

	assertBuilderExercisesBridge(t, rendered)

	// `go vet ./...` covers both production and `_test.go` files,
	// so the rendered mockgen output (which lands as
	// `<source>_mock_test.go`) participates in the check alongside
	// buildergen / repogen's production-shape output.
	cmd := exec.CommandContext(t.Context(), "go", "vet", "./...")
	cmd.Dir = fixtureDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("`go vet ./...` in %s failed: %v\noutput:\n%s\nrendered files: %v",
			fixtureDir, err, out, rendered)
	}
}

// assertBuilderExercisesBridge guards against a regression where
// the rendered output is trivially empty (every field filtered
// out upstream) or where a cross-package reference loses its
// translation. A vacuous Builder would compile but defeat the
// compile check's purpose: the test exists to prove the bridge's
// scalar / well-known / optional / nested composition rules and
// the cross-package proto-qualifier→Go-import translation all
// produced compilable Go, which can only happen when each
// plugin's output references the bridge-translated identifiers
// and import paths.
func assertBuilderExercisesBridge(t *testing.T, rendered []string) {
	t.Helper()
	body := concatBodies(t, rendered)
	for _, want := range []string{
		// buildergen exercises bridge-stamped go.name (snake_case
		// proto field → PascalCase Go identifier) on the export
		// filter, setter names, and composite keys; the optional-
		// wrap + well-known rules surface through the Age and
		// CreatedAt field types; the nested-message reference
		// renders through the underscore-joined Go form.
		"type UserBuilder struct",
		"func (b *UserBuilder) WithName(value string)",
		"func (b *UserBuilder) WithAge(value *int32)",
		"func (b *UserBuilder) WithCreatedAt(value *timestamppb.Timestamp)",
		"func (b *UserBuilder) WithProfileRef(value User_Profile)",
		`"google.golang.org/protobuf/types/known/timestamppb"`,
		// repogen produces the canonical CRUD interface and
		// struct against the proto-derived User type. Same-package
		// references elide the qualifier under the bridge-
		// translated import path.
		"type UserRepository interface",
		"type UserRepo struct",
		"Get(ctx context.Context, id string) (*User, error)",
		// mockgen produces test-package mocks referencing
		// proto-derived request/response types via the bridge-
		// translated Go import path of the source package.
		"package buildfixture_test",
		"type UserServiceMock struct",
		"func (m *UserServiceMock) GetUser(arg0 buildfixture.GetUserRequest) buildfixture.User",
		`"go.thesmos.sh/eidos/eidostest/protopipe/buildfixture"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered output missing %q\n--- rendered ---\n%s", want, body)
		}
	}
}

// concatBodies reads every rendered file and joins the bytes
// for substring matching. The compile-check fixture produces a
// single rendered file today; concatenation keeps the helper
// resilient to a future fixture that splits output across files.
func concatBodies(t *testing.T, paths []string) string {
	t.Helper()
	var b strings.Builder
	for _, p := range paths {
		body, err := os.ReadFile(p) //nolint:gosec // path comes from rendered-file enumeration, not user input.
		if err != nil {
			t.Fatalf("reading rendered %s: %v", p, err)
		}
		b.Write(body)
	}
	return b.String()
}

// buildFixtureDir resolves the absolute path of the buildfixture
// package directory through [runtime.Caller]. The package lives
// outside the testdata/ tree so the hand-written stubs compile
// under the eidos module's standard `go build ./...` invocation.
func buildFixtureDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "buildfixture")
}

// writeRenderedFilesInto materialises every file captured by the
// in-memory sink into dir, registering a cleanup hook that
// removes them after the test. Returns the absolute paths of the
// files written so a build failure can surface what landed where.
func writeRenderedFilesInto(t *testing.T, mem *sink.Memory, dir string) []string {
	t.Helper()
	written := make([]string, 0, mem.Len())
	for target, body := range mem.Files() {
		path := filepath.Join(dir, target.Filename)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatalf("writing rendered file %s: %v", path, err)
		}
		written = append(written, path)
	}
	t.Cleanup(func() {
		for _, p := range written {
			_ = os.Remove(p)
		}
	})
	return written
}
