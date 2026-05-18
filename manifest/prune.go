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
// current treats every prev output owned by pipelineID as stale.
//
// pipelineID scopes the diff: only prev entries whose
// [Output.PipelineID] matches the supplied id participate. Other
// pipelines' entries are out of bounds — they're produced by a
// different pipeline that prune has no authority over. An empty
// pipelineID disables the scope filter (every prev entry is in
// scope) — the legacy single-pipeline behaviour for callers that
// have not migrated to multi-pipeline awareness.
func Prune(prev, current *Manifest, pipelineID string) []emit.Target {
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
		if pipelineID != "" && o.PipelineID != pipelineID {
			continue
		}
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
