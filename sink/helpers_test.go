// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/emit"
)

// assertNoError fails the test if err is non-nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// targetAt returns a populated [emit.Target] used by sink tests
// that need a routable destination.
func targetAt(dir, file string) emit.Target {
	return emit.Target{Dir: dir, Filename: file, Package: "x"}
}

// errWriter returns the configured error from every Write call.
// Used by stdout-sink tests to exercise the error-propagation path
// without touching the real os.Stdout.
type errWriter struct{ err error }

func (e *errWriter) Write([]byte) (int, error) { return 0, e.err }

// boundedWriter accepts the first `succeed` writes and fails the
// rest. Used by stdout-sink tests to drive specific error paths
// (header succeeds, body or trailing newline fails).
type boundedWriter struct {
	succeed int
}

func (b *boundedWriter) Write(p []byte) (int, error) {
	if b.succeed <= 0 {
		return 0, errors.New("bounded: refused")
	}
	b.succeed--
	return len(p), nil
}
