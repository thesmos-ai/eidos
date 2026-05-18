// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package manifest

import (
	"strings"

	"go.thesmos.sh/eidos/emit"
)

// Prune returns the list of orphan [Output] entries in prev —
// entries the current run had authority over but did NOT
// re-emit. The CLI's "eidos prune" command deletes the
// corresponding files; the "eidos check" gate reports them as
// drift without deleting.
//
// An entry is an orphan iff all three conditions hold:
//
//  1. [Output.PipelineID] equals pipelineID. Other pipelines'
//     entries are off-limits; their owning pipeline manages
//     their lifecycle.
//  2. The entry's [emit.Target.ImportPath] (or its `<pkg>_test`
//     auto-shift variant) is in scope — i.e. the current
//     pipeline loaded that source package. Narrow runs that did
//     not load the source package don't get to call its outputs
//     orphans.
//  3. The entry's Target is NOT in the emitted set — i.e. the
//     current run did not write to that target. Targets that
//     the current run produced are not orphans.
//
// Order is preserved from prev so iteration is deterministic
// for the same input. A nil prev manifest, an empty pipelineID,
// a nil scope, or a nil emitted set all return nil (nothing
// identifiable as an orphan).
func Prune(
	prev *Manifest,
	emitted map[emit.Target]struct{},
	scope map[string]struct{},
	pipelineID string,
) []Output {
	if prev == nil || pipelineID == "" || scope == nil || emitted == nil {
		return nil
	}
	var out []Output
	for _, o := range prev.Outputs {
		if o.PipelineID != pipelineID {
			continue
		}
		if !inScope(o.Target.ImportPath, scope) {
			continue
		}
		if _, claimed := emitted[o.Target]; claimed {
			continue
		}
		out = append(out, o)
	}
	return out
}

// inScope reports whether the supplied import path lives in the
// scope set — exact match plus the framework's `<pkg>_test`
// auto-shift variant. Empty input is out of scope.
func inScope(path string, scope map[string]struct{}) bool {
	if path == "" {
		return false
	}
	if _, ok := scope[path]; ok {
		return true
	}
	if stripped, ok := strings.CutSuffix(path, "_test"); ok {
		if _, hit := scope[stripped]; hit {
			return true
		}
	}
	return false
}
