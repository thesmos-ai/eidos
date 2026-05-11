// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package testpipe

import (
	"bytes"
	"testing"

	"go.thesmos.sh/eidos/emit"
)

// FileAssertion accumulates per-file expectations against a captured
// rendered file. Each assertion failure calls t.Errorf so chained
// expectations all report; tests that need stop-on-first-failure
// semantics insert a [testing.TB.FailNow] between assertions.
//
// FileAssertion values are produced by [Pipeline.AssertFile] /
// [Pipeline.AssertFileInDir] — never constructed directly.
type FileAssertion struct {
	t      testing.TB
	target emit.Target
	body   []byte
}

// Target returns the [emit.Target] the file was rendered against.
// Used by tests that want to assert on routing metadata in addition
// to content.
func (a *FileAssertion) Target() emit.Target { return a.target }

// Bytes returns a copy of the file's rendered content. Mutations on
// the returned slice do not affect the assertion's state or later
// chained calls.
func (a *FileAssertion) Bytes() []byte {
	dup := make([]byte, len(a.body))
	copy(dup, a.body)
	return dup
}

// String returns the file's content as a string. Convenient for
// debugging tests via Logf("%s", a.String()).
func (a *FileAssertion) String() string { return string(a.body) }

// Contains fails the test (without stopping) when substr does not
// appear in the file's content.
func (a *FileAssertion) Contains(substr string) *FileAssertion {
	a.t.Helper()
	if !bytes.Contains(a.body, []byte(substr)) {
		a.t.Errorf("testpipe: file %s does not contain %q\n----- file body -----\n%s\n----- end -----",
			a.target.JoinPath(), substr, a.body)
	}
	return a
}

// NotContains fails the test (without stopping) when substr appears
// in the file's content.
func (a *FileAssertion) NotContains(substr string) *FileAssertion {
	a.t.Helper()
	if bytes.Contains(a.body, []byte(substr)) {
		a.t.Errorf("testpipe: file %s unexpectedly contains %q\n----- file body -----\n%s\n----- end -----",
			a.target.JoinPath(), substr, a.body)
	}
	return a
}

// Equals fails the test (without stopping) when the file's content
// is not byte-identical to expected. The failure message uses the
// captured target's [emit.Target.JoinPath] so the user can identify
// the file at a glance.
func (a *FileAssertion) Equals(expected string) *FileAssertion {
	a.t.Helper()
	if string(a.body) != expected {
		a.t.Errorf(
			"testpipe: file %s did not match\n----- got -----\n%s\n----- want -----\n%s\n----- end -----",
			a.target.JoinPath(), a.body, expected,
		)
	}
	return a
}

// EqualsBytes is the bytes-typed companion to [FileAssertion.Equals].
func (a *FileAssertion) EqualsBytes(expected []byte) *FileAssertion {
	a.t.Helper()
	if !bytes.Equal(a.body, expected) {
		a.t.Errorf(
			"testpipe: file %s did not match\n----- got -----\n%s\n----- want -----\n%s\n----- end -----",
			a.target.JoinPath(), a.body, expected,
		)
	}
	return a
}
