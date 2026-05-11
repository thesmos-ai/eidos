// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"sync"

	"go.thesmos.sh/eidos/emit"
)

// Writer is the per-output-file buffer the backend assembles
// content into during template rendering. Each Writer owns one
// [ImportSet] (the file's import block) and an internal list of
// keyed contributions that the writer concatenates in deterministic
// order when [Writer.Body] is called.
//
// Two append modes share the same backing store:
//
//   - [Writer.Append] adds a sequential contribution. The writer
//     auto-assigns a monotonically-increasing internal key, so a
//     stream of sequential Appends concatenates in append order
//     when finalised. Use this for the default within-file-sequential
//     rendering model from spec §18.
//   - [Writer.AppendKeyed] adds a contribution under a caller-supplied
//     key. On finalisation, keyed contributions sort by key
//     alphabetically — independent goroutines may dispatch
//     concurrent AppendKeyed calls and the final output is
//     deterministic regardless of mutex-acquisition order.
//
// Sequential contributions always render before keyed contributions
// (the internal sequential keys sort lexicographically below the
// "user:" namespace of keyed contributions). Most callers pick one
// mode per Writer; mixing is well-defined but rarely useful.
//
// Writer is safe for concurrent use; the mutex serialises appends
// and finalisation.
type Writer struct {
	target  emit.Target
	imports *ImportSet

	mu      sync.Mutex
	seq     int
	entries []entry
}

// entry is one keyed contribution. The writer concatenates entries
// in lexicographic key order on finalisation.
type entry struct {
	key  string
	body []byte
}

// keyAuto is the sequential-Append key prefix; keyUser is the
// keyed-Append prefix. The two namespaces use distinct leading
// characters so sequential always sorts before keyed regardless of
// the user-supplied key.
const (
	keyAuto = "0:auto:"
	keyUser = "1:user:"
)

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

// Append adds body as a sequential contribution. The writer
// records the call under a monotonically-increasing internal key,
// preserving append order on finalisation. Use this for the
// default within-file-sequential rendering model.
func (w *Writer) Append(body []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	key := fmt.Sprintf("%s%020d", keyAuto, w.seq)
	w.seq++
	w.entries = append(w.entries, copyEntry(key, body))
}

// AppendString is a string-flavoured [Writer.Append] convenience.
func (w *Writer) AppendString(s string) { w.Append([]byte(s)) }

// AppendKeyed adds body under the caller-supplied key. On
// finalisation, keyed contributions sort alphabetically by key, so
// concurrent AppendKeyed calls from independent goroutines produce
// a deterministic body regardless of which goroutine ran first.
//
// Use AppendKeyed when the backend renders entities concurrently
// within a single file; pass the entity's plan position (or
// qualified name) as the key so the final body matches the plan
// order. Sequential and keyed contributions render in two ordered
// groups: sequential first (in [Writer.Append] order), keyed
// second (in alphabetical key order).
func (w *Writer) AppendKeyed(key string, body []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entries = append(w.entries, copyEntry(keyUser+key, body))
}

// copyEntry copies body into a new entry. Defensive copy so later
// caller mutations of body don't affect the writer's recorded
// state.
func copyEntry(key string, body []byte) entry {
	dup := make([]byte, len(body))
	copy(dup, body)
	return entry{key: key, body: dup}
}

// Body returns the concatenated body bytes in key order: sequential
// contributions first (in append order), then keyed contributions
// in alphabetical key order. The returned slice is a fresh
// allocation safe for callers to mutate.
func (w *Writer) Body() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	sorted := make([]entry, len(w.entries))
	copy(sorted, w.entries)
	slices.SortStableFunc(sorted, func(a, b entry) int {
		return cmp.Compare(a.key, b.key)
	})
	var out bytes.Buffer
	for _, e := range sorted {
		out.Write(e.body)
	}
	return out.Bytes()
}

// Len returns the total number of bytes across every recorded
// contribution.
func (w *Writer) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, e := range w.entries {
		n += len(e.body)
	}
	return n
}
