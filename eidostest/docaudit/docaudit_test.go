// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package docaudit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/eidostest/docaudit"
)

// TestAssertEveryMetaKeyDocumented_AllPresent covers the happy
// path: every meta-key literal under the package directory
// appears in doc.go.
func TestAssertEveryMetaKeyDocumented_AllPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "doc.go"), `// Package foo documents foo.bar and foo.baz.
package foo
`)
	writeFile(t, filepath.Join(dir, "meta.go"), `package foo

import "go.thesmos.sh/eidos/core/meta"

var Bar = meta.NewKey("foo.bar", meta.StringParser)
var Baz = meta.EnsureKey("foo.baz", meta.StringParser)
`)
	docaudit.AssertEveryMetaKeyDocumented(t, dir)
}

// TestAssertEveryMetaKeyDocumented_MissingKeyFails pins the
// failure mode: a meta-key literal not mentioned in doc.go
// surfaces as a t.Errorf entry naming the missing key.
func TestAssertEveryMetaKeyDocumented_MissingKeyFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "doc.go"), `// Package foo documents foo.bar only.
package foo
`)
	writeFile(t, filepath.Join(dir, "meta.go"), `package foo

import "go.thesmos.sh/eidos/core/meta"

var Bar = meta.NewKey("foo.bar", meta.StringParser)
var Missing = meta.NewKey("foo.missing", meta.StringParser)
`)
	fake := newFake()
	docaudit.AssertEveryMetaKeyDocumented(fake, dir)
	if !fake.failed {
		t.Fatalf("expected failure for undocumented key")
	}
	joined := strings.Join(fake.errs, "\n")
	if !strings.Contains(joined, "foo.missing") {
		t.Fatalf("error should name the missing key; got %q", joined)
	}
}

// TestAssertEveryMetaKeyDocumented_SkipsDynamicNames covers the
// dynamic-name skip rule: a meta.EnsureKey call whose first
// argument is not a literal string is omitted from the audit,
// since the key resolves through a runtime-composed name the
// caller documents under a namespace prefix.
func TestAssertEveryMetaKeyDocumented_SkipsDynamicNames(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "doc.go"), `// Package foo documents foo.literal under namespace foo.prefix.
package foo
`)
	writeFile(t, filepath.Join(dir, "meta.go"), `package foo

import "go.thesmos.sh/eidos/core/meta"

const prefix = "foo.prefix."

var Literal = meta.NewKey("foo.literal", meta.StringParser)

func DynamicKey(name string) meta.Key[string] {
	return meta.EnsureKey(prefix+name, meta.StringParser)
}
`)
	docaudit.AssertEveryMetaKeyDocumented(t, dir)
}

// writeFile is a tiny helper that writes path with body bytes
// and fails the test on error.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// fakeT is a minimal stand-in for [testing.T] that captures
// Errorf calls without affecting the host test's pass/fail. The
// audit-failure subtest uses it to drive the helper and inspect
// what it would have reported.
type fakeT struct {
	failed bool
	errs   []string
}

// newFake returns a fresh fakeT.
func newFake() *fakeT { return &fakeT{} }

// Helper satisfies the helper-marker on [testing.TB].
func (*fakeT) Helper() {}

// Errorf records a non-fatal failure entry.
func (f *fakeT) Errorf(format string, args ...any) {
	f.failed = true
	f.errs = append(f.errs, fmt.Sprintf(format, args...))
}

// Fatalf records a fatal failure. The audit helper's positive-
// failure subtest never reaches Fatalf in the missing-key path
// (every missing key is a non-fatal Errorf); a Fatalf here is a
// wiring error worth surfacing.
func (f *fakeT) Fatalf(format string, args ...any) {
	f.failed = true
	f.errs = append(f.errs, "FATAL: "+fmt.Sprintf(format, args...))
}
