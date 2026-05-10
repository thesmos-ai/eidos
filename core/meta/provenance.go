// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta

import (
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/core/position"
)

// Provenance is one recorded operation against a metadata key.
//
// A Bag stores at most one Provenance per (name, authority) pair —
// later writes at the same authority replace the earlier record.
// [Bag.History] returns the per-authority records for a name in
// ascending-authority order; [Bag.Winning] returns the entry the
// resolution algorithm picks.
//
// CoveredBy is empty for direct entries and names the ancestor key
// when this entry was produced by a prefix tombstone (e.g. a
// tombstone on "shape.writer" covering "shape.writer.method"). The
// field lets `eidos explain` render "tombstoned via -gen:shape
// writer" rather than fabricating a direct entry at the descendant.
type Provenance struct {
	Authority Authority
	Tombstone bool
	Value     any
	SetBy     string
	Pos       position.Pos
	CoveredBy string
}

// String renders p for human-readable trail output.
//
// Format:
//
//	plugin   set true              by writershape at user.go:42:1
//	directive tombstone           via -gen:shape writer at file.go:18:1
//	directive tombstone (covers shape.writer.method) by directive at file.go:18:1
func (p Provenance) String() string {
	var b strings.Builder
	b.WriteString(p.Authority.String())
	b.WriteByte(' ')
	if p.Tombstone {
		b.WriteString("tombstone")
	} else {
		fmt.Fprintf(&b, "set %v", p.Value)
	}
	if p.CoveredBy != "" {
		fmt.Fprintf(&b, " (covers via %s)", p.CoveredBy)
	}
	if p.SetBy != "" {
		fmt.Fprintf(&b, " by %s", p.SetBy)
	}
	if !p.Pos.IsZero() {
		fmt.Fprintf(&b, " at %s", p.Pos)
	}
	return b.String()
}
