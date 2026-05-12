package blog

import "io"

// LineWriter is the canonical writer-shape fixture entity. Its
// method set carries the exact `Write(p []byte) (n int, err error)`
// signature the shape detector targets; the embedded io.Closer
// exercises the backend's embed rendering path.
//
// LineWriter intentionally carries no `+gen:*` directives — the
// shape detector reaches it purely through signature matching, and
// the heuristic-vs-directive split is verified by negative-override
// tests that suppress detection here.
type LineWriter struct {
	io.Closer

	// Lines buffers the written lines so the cross-cutting weavers
	// have non-trivial body content to wrap.
	Lines []string
}

// Write records p as a new line and reports the byte count back to
// the caller. Signature matches io.Writer.
func (w *LineWriter) Write(p []byte) (n int, err error) {
	w.Lines = append(w.Lines, string(p))
	return len(p), nil
}
