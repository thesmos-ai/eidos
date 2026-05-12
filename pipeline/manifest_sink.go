// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package pipeline

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"sync"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/manifest"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// recordingSink wraps the user-supplied [sink.Sink] and captures
// every Write into an internal map keyed by [emit.Target]. At run
// end the pipeline reads the captured set, hashes each payload, and
// writes the resulting [manifest.Manifest] to disk.
//
// The wrapper preserves write semantics — the inner sink still
// receives every payload — and is concurrent-safe so parallel
// backend file rendering can dispatch without coordination.
type recordingSink struct {
	mu    sync.Mutex
	inner sink.Sink
	files map[emit.Target][]byte
}

// newRecordingSink returns a recordingSink wrapping inner.
func newRecordingSink(inner sink.Sink) *recordingSink {
	return &recordingSink{inner: inner, files: map[emit.Target][]byte{}}
}

// Write delegates to the inner sink and, on success, captures a
// copy of the payload keyed by target for later manifest assembly.
// Write errors from the inner sink propagate verbatim; no capture
// happens on failure.
func (r *recordingSink) Write(target emit.Target, body []byte) error {
	if err := r.inner.Write(target, body); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	dup := make([]byte, len(body))
	copy(dup, body)
	r.files[target] = dup
	return nil
}

// asManifest produces a [manifest.Manifest] from every captured
// write. Each entry's Hash is the SHA-256 hex digest of the
// payload prefixed with "sha256:". Plugins is the list of plugin
// names that produced entities routed to this target — derived
// from each entity's [emit.Node.SetBy] via s.Emit().ByTarget(),
// ordered by `order` (the pipeline's registration order) so two
// runs over the same input produce byte-identical manifests.
// ResolvedLayout is the per-output observability block the Layout
// phase published for each routed target.
//
// A *Pipeline is required (the recording sink only exists as part
// of a Pipeline.Run). When Layout composed at least one Target
// during the run, every captured Target must match a composed
// entry — a mismatch means the backend wrote to a Target Layout
// didn't see, which is a framework bug and surfaces as an
// Internal diagnostic. When Layout composed nothing (a synthetic
// run with no routable decls), the strict-attribution check
// stands down: nothing is being violated, just no observability
// metadata is available to attach. The backend write still
// records into the manifest with the hash and plugin attribution
// it has.
//
// The same per-target attribution flows into the rendered file's
// `Plugins:` header (see backend/golang.pluginsFor), so manifest
// and on-disk provenance agree on which plugins contributed to
// each output.
func (r *recordingSink) asManifest(
	runID string,
	s *store.Store,
	order []string,
	p *Pipeline,
) *manifest.Manifest {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Sort captured Targets so the manifest's Outputs slice is
	// stable across runs. Map iteration order would otherwise
	// reshuffle every entry on each run, churning the manifest's
	// on-disk bytes even when no output content changed.
	targets := make([]emit.Target, 0, len(r.files))
	for target := range r.files {
		targets = append(targets, target)
	}
	slices.SortFunc(targets, func(a, b emit.Target) int {
		if c := cmp.Compare(a.Dir, b.Dir); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Filename, b.Filename); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Package, b.Package); c != 0 {
			return c
		}
		return cmp.Compare(a.ImportPath, b.ImportPath)
	})

	layoutRouted := p.hasLayoutActivity()
	m := manifest.New(runID)
	for _, target := range targets {
		body := r.files[target]
		sum := sha256.Sum256(body)
		out := manifest.Output{
			Target:  target,
			Plugins: pluginsForTarget(s, target, order),
			Hash:    "sha256:" + hex.EncodeToString(sum[:]),
		}
		if rl, ok := p.resolvedLayoutFor(target); ok {
			rl := rl // local for pointer
			out.ResolvedLayout = &rl
		} else if layoutRouted {
			p.diag.Internalf(position.Pos{},
				"manifest: backend wrote to %+v without Layout-composed attribution",
				target)
		}
		m.Add(out)
	}
	return m
}

// pluginsForTarget returns the unique plugin identifiers that
// produced content routed to target, ordered by the pipeline's
// registration order. Both top-level entity attribution
// ([emit.Node.SetBy]) and slot-contribution provenance
// ([emit.Provenance.SetBy], including method-body weaver
// contributions) participate. The empty string ("unattributed")
// drops out unconditionally.
//
// Falls back to the full `order` slice when no entity carries
// attribution — the test-fixture case where entities are
// hand-built outside the builder — so the manifest's per-output
// Plugins list stays consistent with the rendered file's
// `Plugins:` header in either regime.
func pluginsForTarget(s *store.Store, target emit.Target, order []string) []string {
	contributed := map[string]bool{}
	for _, e := range s.Emit().ByTarget().Get(target) {
		collectTargetContributors(e, contributed)
	}
	if len(contributed) == 0 {
		return append([]string(nil), order...)
	}
	out := make([]string, 0, len(contributed))
	for _, name := range order {
		if contributed[name] {
			out = append(out, name)
		}
	}
	return out
}

// collectTargetContributors walks n's own SetBy plus the SetBy of
// every Provenance stamped on n's slots, recursing one level into
// the methods of method-bearing hosts. Mirrors the per-target
// collector the backend's header renderer uses so manifest and
// header report the same per-target plugin set.
func collectTargetContributors(n emit.Node, out map[string]bool) {
	if name := strings.TrimSpace(n.SetBy()); name != "" {
		out[name] = true
	}
	if host, ok := n.(slotEnumerator); ok {
		for _, slot := range host.SlotsByName() {
			for _, prov := range slot.ProvenanceList {
				if name := strings.TrimSpace(prov.SetBy); name != "" {
					out[name] = true
				}
			}
		}
	}
	switch host := n.(type) {
	case *emit.Struct:
		for _, m := range host.Methods {
			collectTargetContributors(m, out)
		}
	case *emit.Interface:
		for _, m := range host.Methods {
			collectTargetContributors(m, out)
		}
	case *emit.Alias:
		for _, m := range host.Methods {
			collectTargetContributors(m, out)
		}
	}
}

// slotEnumerator is the read-only counterpart of the (unexported)
// slotMap embedded by every emit host.
type slotEnumerator interface {
	SlotsByName() map[string]*emit.Slot
}
