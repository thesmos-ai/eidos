// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer

import (
	"bytes"
	"sync"

	"go.thesmos.sh/eidos/emit"
)

// Writer is the per-output-file buffer the backend assembles
// content into during template rendering. Each Writer owns one
// [ImportSet] (the file's import block) and a body buffer that
// receives the rendered output of every emit entity routed to the
// file's [emit.Target].
//
// Writers are safe for concurrent use: backend code may dispatch
// per-entity rendering goroutines that share one Writer per file.
// The mutex serialises body appends while still allowing parallel
// rendering of distinct files.
//
// The zero value is unusable; construct via [New].
type Writer struct {
	target  emit.Target
	imports *ImportSet
	mu      sync.Mutex
	body    bytes.Buffer
}

// New returns a Writer for target. Pass nil for derive to use
// [DefaultAlias] inside the file's [ImportSet]; pass a language-
// specific function to control aliasing.
func New(target emit.Target, derive AliasFunc) *Writer {
	return &Writer{
		target:  target,
		imports: NewImportSet(derive),
	}
}

// Target returns the [emit.Target] this Writer assembles output
// for.
func (w *Writer) Target() emit.Target { return w.target }

// Imports returns the file's [ImportSet]. The backend's template
// func-map binds `imp` to [ImportSet.Imp]; template authors call
// `imp` to record an external path and receive the local alias.
func (w *Writer) Imports() *ImportSet { return w.imports }

// Append concatenates p to the body buffer. Calls are atomic; two
// concurrent Append goroutines never interleave their bytes.
func (w *Writer) Append(p []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.body.Write(p)
}

// AppendString is a string-flavoured [Writer.Append] convenience.
func (w *Writer) AppendString(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.body.WriteString(s)
}

// Body returns a snapshot of the body buffer as bytes. Subsequent
// Append calls do not affect the returned slice.
func (w *Writer) Body() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]byte, w.body.Len())
	copy(out, w.body.Bytes())
	return out
}

// Len returns the number of bytes currently in the body buffer.
func (w *Writer) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Len()
}
