// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package testpipe

import (
	"sync"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"
)

// frontendName is the [plugin.Plugin.Name] every [FromNodes] frontend
// reports. The pipeline's duplicate-name check rejects two synthetic
// frontends in the same build, which matches the test-shape: one
// synthetic frontend per pipeline.
const frontendName = "testpipe-synthetic"

// FromNodes returns a synthetic [plugin.Frontend] that injects the
// supplied [node.Package] values into the source-side store on Load.
//
// The frontend ignores the load pattern entirely — the test
// constructed the data, so there is no source to parse. Load is
// guarded so repeated invocations (one per pattern supplied to
// [pipeline.Pipeline.Run]) only add the packages once; subsequent
// calls are no-ops. This makes the helper safe to compose into
// pipelines that pass any number of patterns.
//
// Packages with the same path collide on the underlying store's
// duplicate-qname check at Load time; the returned error surfaces as
// a positioned diagnostic on the run's diag sink.
func FromNodes(pkgs ...*node.Package) plugin.Frontend {
	return &nodesFrontend{pkgs: pkgs}
}

// nodesFrontend implements the synthetic frontend produced by
// [FromNodes]. It is concurrent-safe so a pipeline configured with
// parallel frontends (a contrived but legal shape) does not double-
// add or race.
type nodesFrontend struct {
	mu     sync.Mutex
	pkgs   []*node.Package
	loaded bool
}

// Name returns the synthetic frontend's plugin name.
func (*nodesFrontend) Name() string { return frontendName }

// Load adds every package to the store the first time it is called.
// Subsequent calls return nil without touching the store; any
// AddPackage error surfaces as a non-nil return.
func (f *nodesFrontend) Load(_ string, s *store.Store, _ *diag.Sink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.loaded {
		return nil
	}
	for _, p := range f.pkgs {
		if err := s.Nodes().AddPackage(p); err != nil {
			return err
		}
	}
	f.loaded = true
	return nil
}
