// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store

// Query is a typed, deferred query against a slice of T values.
// Predicates added via [Query.Where] accumulate; iteration happens
// only when a terminal call ([Query.Each], [Query.Slice],
// [Query.First], [Query.Count]) is made.
//
// Queries are constructed by a [Reader], not by the [Store]
// directly — a Reader supplies the per-plugin [ReadSet] that the
// terminals update for cache-key derivation.
//
// Query is a value type and may be passed by value or stored by
// pointer; chained calls return a new Query so the original remains
// usable.
type Query[T any] struct {
	source []T
	pred   func(T) bool
	reads  *ReadSet
	tag    string
}

// newQuery constructs a Query from the supplied source slice with
// no predicate. Calls to terminal methods will record the supplied
// tag on the supplied ReadSet (when both are non-nil).
func newQuery[T any](source []T, reads *ReadSet, tag string) *Query[T] {
	return &Query[T]{source: source, reads: reads, tag: tag}
}

// Where adds pred to the query's filter. Multiple Where calls
// compose as a logical AND; a nil pred is treated as a no-op match.
// Where returns a new Query; the receiver is unchanged.
func (q *Query[T]) Where(pred func(T) bool) *Query[T] {
	if pred == nil {
		return q
	}
	combined := pred
	if q.pred != nil {
		prev := q.pred
		combined = func(v T) bool { return prev(v) && pred(v) }
	}
	return &Query[T]{source: q.source, pred: combined, reads: q.reads, tag: q.tag}
}

// Each invokes fn for every item in the source that satisfies the
// accumulated predicate. Iteration order is the source's insertion
// order. Each records the query's tag in the [ReadSet] before
// iterating.
func (q *Query[T]) Each(fn func(T)) {
	q.recordRead()
	for _, item := range q.source {
		if q.pred == nil || q.pred(item) {
			fn(item)
		}
	}
}

// Slice returns the matched items as a new slice in source insertion
// order. Slice records the query's tag in the [ReadSet] before
// materialising.
func (q *Query[T]) Slice() []T {
	q.recordRead()
	out := make([]T, 0, len(q.source))
	for _, item := range q.source {
		if q.pred == nil || q.pred(item) {
			out = append(out, item)
		}
	}
	return out
}

// First returns the first matching item along with true; when no
// item matches, it returns the zero value and false. First records
// the query's tag in the [ReadSet] regardless of whether a match
// was found.
func (q *Query[T]) First() (T, bool) {
	q.recordRead()
	for _, item := range q.source {
		if q.pred == nil || q.pred(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// Count returns the number of items satisfying the accumulated
// predicate. Count records the query's tag in the [ReadSet].
func (q *Query[T]) Count() int {
	q.recordRead()
	if q.pred == nil {
		return len(q.source)
	}
	n := 0
	for _, item := range q.source {
		if q.pred(item) {
			n++
		}
	}
	return n
}

// recordRead appends the query's tag to its [ReadSet]. Every Query
// constructed via [Reader] has a non-nil ReadSet and a non-empty
// tag, so recordRead is unconditional.
func (q *Query[T]) recordRead() {
	q.reads.Record(q.tag)
}
