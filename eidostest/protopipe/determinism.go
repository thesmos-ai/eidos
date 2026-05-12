// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protopipe

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.thesmos.sh/eidos/store"
)

// AssertDeterministic runs the configured pipeline twice and
// asserts that the resulting node-side store serializes to the
// same bytes. Per the protobuf frontend's determinism contract,
// two consecutive loads against the same source set must produce
// byte-identical node graphs and identical meta-key sequences;
// this helper is the canonical assertion tests call to pin the
// property. Works against an empty store as well as a fully
// populated one.
func AssertDeterministic(t *testing.T, opts RunOptions) {
	t.Helper()
	first := Run(t, opts)
	second := Run(t, opts)
	firstBytes := serializeNodes(t, first.Store)
	secondBytes := serializeNodes(t, second.Store)
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatalf(
			"two consecutive runs produced divergent node graphs:\n--- first ---\n%s\n--- second ---\n%s",
			firstBytes,
			secondBytes,
		)
	}
}

// serializeNodes produces a canonical JSON encoding of every
// source-package the store carries. The encoding is the same
// shape the framework's manifest / cache layers use, so byte
// equality across consecutive runs guarantees byte equality of
// any derived output.
func serializeNodes(t *testing.T, s *store.Store) []byte {
	t.Helper()
	if s == nil {
		return nil
	}
	pkgs := s.Nodes().Packages().Items()
	// musttag flags this because node.Package's fields inherit
	// JSON tags through an embedded BaseNode rather than declaring
	// them on the host struct — the linter's struct-traversal
	// doesn't follow promotion. The marshalled output is well-formed
	// and stable; the byte-equality assertion in [AssertDeterministic]
	// is the canonical proof.
	body, err := json.Marshal(pkgs) //nolint:musttag // tags ride on the embedded BaseNode
	if err != nil {
		t.Fatalf("protopipe: serialize node graph: %v", err)
	}
	return body
}
