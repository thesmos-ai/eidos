// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink

import (
	"fmt"
	"io"
	"sync"

	"go.thesmos.sh/eidos/emit"
)

// Stdout is a [Sink] that writes each entry to an [io.Writer] with
// a leading "// === <target-path> ===" header followed by the body
// and a trailing newline. The header makes multi-target output
// scannable when the user pipes it to a pager or saves it to a file.
//
// The constructor accepts any [io.Writer]; callers pass [os.Stdout]
// for production use and a buffer for tests. Concurrent calls
// serialise through a mutex so interleaved writers don't corrupt
// each other's output.
type Stdout struct {
	mu sync.Mutex
	w  io.Writer
}

// NewStdout returns a Stdout sink that writes to w. Pass
// [os.Stdout] for normal use; substitute a buffer (or any other
// [io.Writer]) to capture output in tests.
func NewStdout(w io.Writer) *Stdout {
	return &Stdout{w: w}
}

// Write emits a header + body for target. Returns
// [ErrInvalidTarget] when target.Filename is empty. Write errors
// from the underlying writer propagate verbatim.
func (s *Stdout) Write(target emit.Target, body []byte) error {
	if target.Filename == "" {
		return fmt.Errorf("%w: %+v", ErrInvalidTarget, target)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := target.JoinPath()
	if path == "" {
		// Filename is set but Dir is empty — use just the filename
		// so the header still surfaces a useful identifier.
		path = target.Filename
	}
	if _, err := fmt.Fprintf(s.w, "// === %s ===\n", path); err != nil {
		return fmt.Errorf("sink: write header for %s: %w", path, err)
	}
	if _, err := s.w.Write(body); err != nil {
		return fmt.Errorf("sink: write body for %s: %w", path, err)
	}
	if _, err := s.w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("sink: write trailing newline for %s: %w", path, err)
	}
	return nil
}
