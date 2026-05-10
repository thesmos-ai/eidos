// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package position

import (
	"errors"
	"fmt"
)

// ErrCrossFileRange is returned by [NewRange] and [Range.Union] when
// the operation would produce a Range whose endpoints reference
// different files.
var ErrCrossFileRange = errors.New("position: range cannot span multiple files")

// ErrInvalidRangeOrder is returned by [NewRange] when Start sorts after
// End under [Pos.Compare] — a backwards range is rejected rather than
// silently swapped.
var ErrInvalidRangeOrder = errors.New("position: range Start must not sort after End")

// Range names a contiguous span between two Pos values.
//
// Start is inclusive; End is exclusive — the half-open convention
// used by go/token. Range is a value type; copy freely.
//
// A Range whose Start.File and End.File differ is not well-formed;
// construct ranges via [NewRange] to validate the invariant up front,
// or rely on [Range.Contains] / [Range.Overlaps] which both guard
// against the cross-file case.
type Range struct {
	Start Pos `json:"start"`
	End   Pos `json:"end"`
}

// NewRange returns a validated Range from start and end.
//
// Returns [ErrCrossFileRange] when start.File and end.File differ, and
// [ErrInvalidRangeOrder] when start sorts strictly after end. Both
// errors are wrapped with the offending positions for diagnostic use.
//
// Callers that have already established the invariants (e.g. a parser
// that knows it produced both endpoints from the same scan) may use a
// struct literal directly.
func NewRange(start, end Pos) (Range, error) {
	if start.File != end.File {
		return Range{}, fmt.Errorf("%w: %q vs %q", ErrCrossFileRange, start.File, end.File)
	}
	if start.After(end) {
		return Range{}, fmt.Errorf("%w: %s after %s", ErrInvalidRangeOrder, start, end)
	}
	return Range{Start: start, End: end}, nil
}

// IsZero reports whether r carries no positional information.
func (r Range) IsZero() bool {
	return r.Start.IsZero() && r.End.IsZero()
}

// Contains reports whether p falls within r.
//
// The check uses the half-open convention: r.Start is inclusive,
// r.End is exclusive. A zero range contains no positions; a position
// in a different file from r.Start is not contained.
func (r Range) Contains(p Pos) bool {
	if r.IsZero() {
		return false
	}
	if r.Start.File != p.File {
		return false
	}
	return r.Start.Compare(p) <= 0 && p.Compare(r.End) < 0
}

// String returns a human-readable rendering of r.
//
// When r is zero, the result is empty. When Start equals End, the
// result is the same as Start.String(). Otherwise the result is
// "start-end" using each Pos's own String form.
func (r Range) String() string {
	if r.IsZero() {
		return ""
	}
	if r.Start == r.End {
		return r.Start.String()
	}
	return r.Start.String() + "-" + r.End.String()
}

// Union returns the smallest Range that contains both r and other.
//
// The two ranges must reference the same file; [ErrCrossFileRange] is
// returned otherwise. A zero range is treated as the identity: unioning
// with a zero range yields the non-zero one. Unioning two zero ranges
// yields a zero range. The result's Start is the earlier of the two
// starts; End is the later of the two ends.
func (r Range) Union(other Range) (Range, error) {
	switch {
	case r.IsZero():
		return other, nil
	case other.IsZero():
		return r, nil
	case r.Start.File != other.Start.File:
		return Range{}, fmt.Errorf("%w: %q vs %q", ErrCrossFileRange, r.Start.File, other.Start.File)
	}
	start := r.Start
	if other.Start.Before(start) {
		start = other.Start
	}
	end := r.End
	if other.End.After(end) {
		end = other.End
	}
	return Range{Start: start, End: end}, nil
}

// Overlaps reports whether r and other share at least one position.
//
// Overlap uses the half-open convention consistent with [Range.Contains]:
// touching but non-overlapping ranges (one's End equal to the other's
// Start) do not overlap. Zero ranges and ranges in different files
// never overlap.
func (r Range) Overlaps(other Range) bool {
	if r.IsZero() || other.IsZero() {
		return false
	}
	if r.Start.File != other.Start.File {
		return false
	}
	return r.Start.Before(other.End) && other.Start.Before(r.End)
}
