// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer

import (
	"cmp"
	"slices"

	"go.thesmos.sh/eidos/emit"
)

// SortSlotByPlan returns the items in slot reordered to match the
// supplied plan-position map. Items contributed by a plugin with a
// lower planPosition come first; ties (same plugin or both
// unmapped) preserve the slot's recorded insertion order.
//
// Sequential pipeline runs already append slot items in plan order
// — sequential append-then-render is already deterministic. The
// helper exists for parallel-execution runs where two generators
// in the same bucket race to append; this re-sort restores
// determinism for backends that observe the slot post-generator.
//
// Items whose [emit.Provenance.SetBy] is not present in
// planPosition sort after every mapped item, preserving their
// relative insertion order. This matches the spec's "ties by plugin
// name" rule when the plan happens to assign equal positions and
// keeps unattributed contributions at the end of the slot.
func SortSlotByPlan(s *emit.Slot, planPosition map[string]int) []emit.Node {
	if s == nil || s.Len() == 0 {
		return nil
	}
	type entry struct {
		idx  int
		pos  int
		item emit.Node
	}
	unset := len(planPosition) + 1
	pairs := make([]entry, s.Len())
	for i := 0; i < s.Len(); i++ {
		setBy := s.ProvenanceAt(i).SetBy
		pos, mapped := planPosition[setBy]
		if !mapped {
			pos = unset
		}
		pairs[i] = entry{idx: i, pos: pos, item: s.At(i)}
	}
	slices.SortStableFunc(pairs, func(a, b entry) int {
		if c := cmp.Compare(a.pos, b.pos); c != 0 {
			return c
		}
		return cmp.Compare(a.idx, b.idx)
	})
	out := make([]emit.Node, len(pairs))
	for i, p := range pairs {
		out[i] = p.item
	}
	return out
}
