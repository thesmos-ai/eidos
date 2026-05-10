// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package position

import (
	"cmp"
	"strconv"
	"strings"
)

// Pos identifies a single byte location in a source file.
//
// Convention follows go/token: Line and Column are 1-based; Offset is
// the 0-based byte offset from file start. Any field set to its zero
// value means "unknown" for that dimension. A populated File without a
// Line still names the file — useful for diagnostics that target a
// whole file rather than a specific line.
//
// Pos is a value type; copy freely.
type Pos struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// At returns a Pos for file at the given 1-based line and column,
// with no byte offset recorded. It is the most common way to
// construct a Pos in code — diagnostic emitters, test fixtures, and
// any consumer that does not need offset precision should prefer it
// over struct-literal construction.
func At(file string, line, column int) Pos {
	return Pos{File: file, Line: line, Column: column}
}

// AtOffset returns a Pos with byte-offset precision. Use this when
// the originating source supplies offset information (e.g. a Go
// frontend translating from go/token); use [At] when offset is
// irrelevant.
func AtOffset(file string, line, column, offset int) Pos {
	return Pos{File: file, Line: line, Column: column, Offset: offset}
}

// IsZero reports whether p carries no positional information.
//
// A position with only File set is not zero — the file is known even
// if no line / column is.
func (p Pos) IsZero() bool {
	return p.File == "" && p.Line == 0 && p.Column == 0 && p.Offset == 0
}

// String returns a human-readable rendering of p.
//
// The format is "file:line:column" when all three are populated;
// "file:line" when Column is unknown; "file" when only File is set;
// the empty string when p is the zero value.
func (p Pos) String() string {
	if p.IsZero() {
		return ""
	}
	var b strings.Builder
	b.WriteString(p.File)
	if p.Line > 0 {
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(p.Line))
		if p.Column > 0 {
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(p.Column))
		}
	}
	return b.String()
}

// Compare returns -1, 0, or +1 to order p relative to other.
//
// Files compare by string. Within the same file, positions order by
// Line, then Column, then Offset. The result is total: any two Pos
// values have a defined ordering, suitable for sorting.
func (p Pos) Compare(other Pos) int {
	if c := cmp.Compare(p.File, other.File); c != 0 {
		return c
	}
	if c := cmp.Compare(p.Line, other.Line); c != 0 {
		return c
	}
	if c := cmp.Compare(p.Column, other.Column); c != 0 {
		return c
	}
	return cmp.Compare(p.Offset, other.Offset)
}

// Equal reports whether p and other refer to the same byte location.
// It is a thin wrapper over [Pos.Compare] for readability at call
// sites; the value-type equality operator (==) is equivalent.
func (p Pos) Equal(other Pos) bool { return p.Compare(other) == 0 }

// Before reports whether p sorts strictly earlier than other under
// [Pos.Compare]. Useful when the intent is "is this position before
// that one" rather than the lower-level integer comparison.
func (p Pos) Before(other Pos) bool { return p.Compare(other) < 0 }

// After reports whether p sorts strictly later than other under
// [Pos.Compare].
func (p Pos) After(other Pos) bool { return p.Compare(other) > 0 }

// IsSynthetic reports whether p was produced by [Synthetic] (or follows
// the same File="<tag>" convention). Renderers and IDE-aware tooling
// use this to skip click-through behaviour that would attempt to open
// a path that doesn't exist on disk.
func (p Pos) IsSynthetic() bool {
	return strings.HasPrefix(p.File, "<") && strings.HasSuffix(p.File, ">") && len(p.File) >= 2
}

// Synthetic returns a Pos for a purely-generated artifact — code that
// has no real source file. The tag is wrapped in angle brackets so
// downstream renderers can distinguish synthetic positions from real
// paths: Synthetic("repogen") yields File="<repogen>".
//
// Pass a short, meaningful identifier — typically the producing plugin's
// name, or "generated" for unattributed output. [Pos.IsSynthetic]
// reports whether a Pos was produced this way.
func Synthetic(tag string) Pos {
	return Pos{File: "<" + tag + ">"}
}
