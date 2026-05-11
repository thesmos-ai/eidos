// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest

import "go.thesmos.sh/eidos/emit"

// Prune returns the list of [emit.Target] values present in prev
// but not in current — the files the previous run produced that
// the current configuration no longer claims. The CLI's "eidos
// prune" command deletes the corresponding files; the "eidos
// check" gate reports them as drift without deleting.
//
// Order is preserved from prev so iteration is deterministic for
// the same input. Nil manifests are treated as empty: a nil prev
// returns no candidates (no previous run to diff against); a nil
// current treats every prev output as stale.
func Prune(prev, current *Manifest) []emit.Target {
	if prev == nil || len(prev.Outputs) == 0 {
		return nil
	}
	claimed := make(map[emit.Target]struct{}, currentLen(current))
	if current != nil {
		for _, o := range current.Outputs {
			claimed[o.Target] = struct{}{}
		}
	}
	out := make([]emit.Target, 0, len(prev.Outputs))
	for _, o := range prev.Outputs {
		if _, ok := claimed[o.Target]; !ok {
			out = append(out, o.Target)
		}
	}
	return out
}

// currentLen returns the output count of m, treating nil as zero.
// Used to size the claimed-set map; the helper avoids a nil-deref
// branch inline at the call site.
func currentLen(m *Manifest) int {
	if m == nil {
		return 0
	}
	return len(m.Outputs)
}
