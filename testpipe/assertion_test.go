// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package testpipe_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/testpipe"
	"go.thesmos.sh/eidos/emit"
)

// fileAssertion drives one captured file through a freshly-built
// pipeline and returns the FileAssertion bound to the supplied TB.
func fileAssertion(tb testing.TB, body string) *testpipe.FileAssertion {
	tb.Helper()
	tgt := emit.Target{Dir: "out", Filename: "f.go", Package: "out"}
	p := testpipe.New(tb).
		WithFrontend(testpipe.FromNodes()).
		WithBackend(&stubBackend{
			name:   "stub-be",
			lang:   "stub",
			writes: map[emit.Target][]byte{tgt: []byte(body)},
		}).
		Build().
		Run()
	return p.AssertFile("f.go")
}

func TestFileAssertion_Target(t *testing.T) {
	t.Parallel()

	t.Run("returns the target the file was rendered against", func(t *testing.T) {
		t.Parallel()
		a := fileAssertion(t, "hello")
		if a.Target().Filename != "f.go" {
			t.Fatalf("target mismatch: %+v", a.Target())
		}
	})
}

func TestFileAssertion_Bytes(t *testing.T) {
	t.Parallel()

	t.Run("returns a copy independent of the assertion's storage", func(t *testing.T) {
		t.Parallel()
		a := fileAssertion(t, "hello")
		got := a.Bytes()
		if string(got) != "hello" {
			t.Fatalf("Bytes returned wrong content: %q", got)
		}
		got[0] = 'X'
		if string(a.Bytes()) != "hello" {
			t.Fatalf("mutating the returned slice must not affect the assertion")
		}
	})
}

func TestFileAssertion_String(t *testing.T) {
	t.Parallel()

	t.Run("returns the file content as a string", func(t *testing.T) {
		t.Parallel()
		a := fileAssertion(t, "hello")
		if a.String() != "hello" {
			t.Fatalf("String returned %q, want %q", a.String(), "hello")
		}
	})
}

func TestFileAssertion_Contains(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when the substring is present", func(t *testing.T) {
		t.Parallel()
		fileAssertion(t, "alpha beta gamma").Contains("beta")
	})

	t.Run("Errorf when the substring is missing", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		fileAssertion(fake, "alpha").Contains("beta")
		if !fake.Failed() {
			t.Fatalf("expected Errorf on missing substring")
		}
		if !strings.Contains(strings.Join(fake.errs, "\n"), "beta") {
			t.Fatalf("error should mention the missing substring; got %v", fake.errs)
		}
	})

	t.Run("returns the assertion for chaining", func(t *testing.T) {
		t.Parallel()
		a := fileAssertion(t, "hello world")
		if a.Contains("hello") != a {
			t.Fatalf("Contains should return its receiver for chaining")
		}
	})
}

func TestFileAssertion_NotContains(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when the substring is absent", func(t *testing.T) {
		t.Parallel()
		fileAssertion(t, "alpha gamma").NotContains("beta")
	})

	t.Run("Errorf when the substring is present", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		fileAssertion(fake, "alpha beta gamma").NotContains("beta")
		if !fake.Failed() {
			t.Fatalf("expected Errorf on present substring")
		}
	})

	t.Run("returns the assertion for chaining", func(t *testing.T) {
		t.Parallel()
		a := fileAssertion(t, "hello")
		if a.NotContains("world") != a {
			t.Fatalf("NotContains should return its receiver for chaining")
		}
	})
}

func TestFileAssertion_Equals(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when content is byte-identical", func(t *testing.T) {
		t.Parallel()
		fileAssertion(t, "exact").Equals("exact")
	})

	t.Run("Errorf when content differs", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		fileAssertion(fake, "got").Equals("want")
		if !fake.Failed() {
			t.Fatalf("expected Errorf on mismatched content")
		}
	})

	t.Run("returns the assertion for chaining", func(t *testing.T) {
		t.Parallel()
		a := fileAssertion(t, "hello")
		if a.Equals("hello") != a {
			t.Fatalf("Equals should return its receiver for chaining")
		}
	})
}

func TestFileAssertion_EqualsBytes(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when content is byte-identical", func(t *testing.T) {
		t.Parallel()
		fileAssertion(t, "exact").EqualsBytes([]byte("exact"))
	})

	t.Run("Errorf when content differs", func(t *testing.T) {
		t.Parallel()
		fake := newFakeT()
		fileAssertion(fake, "got").EqualsBytes([]byte("want"))
		if !fake.Failed() {
			t.Fatalf("expected Errorf on mismatched bytes")
		}
	})

	t.Run("returns the assertion for chaining", func(t *testing.T) {
		t.Parallel()
		a := fileAssertion(t, "hello")
		if a.EqualsBytes([]byte("hello")) != a {
			t.Fatalf("EqualsBytes should return its receiver for chaining")
		}
	})
}
