// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo_test

import (
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/bridge/protogo"
	"go.thesmos.sh/eidos/eidostest/protopipe"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/mockgen"
	"go.thesmos.sh/eidos/sink"
)

// TestRender_BridgeAffectsGoOutput pins the end-to-end render
// effect: with the protogo bridge registered, mockgen's emitted
// mock for a proto service references the Go-side translated
// type names rather than the bare proto-source names. The
// rendered output is captured through the in-memory sink and
// grep-checked for the expected Go-side identifiers.
func TestRender_BridgeAffectsGoOutput(t *testing.T) {
	t.Parallel()

	t.Run("mockgen output threads through protogo's translation tables", func(t *testing.T) {
		t.Parallel()
		mem := sink.NewMemory()
		root := servicesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir:  root,
			Annotators: []plugin.Annotator{protogo.New()},
			Generators: []plugin.Generator{mockgen.New()},
			Backend:    backend_golang.New(),
			Sink:       mem,
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		body := collectSinkBody(mem)
		if !strings.Contains(body, "GreeterMock") {
			t.Fatalf("expected GreeterMock in rendered output; got:\n%s", body)
		}
		if !strings.Contains(body, "HelloRequest") {
			t.Fatalf("expected HelloRequest reference; got:\n%s", body)
		}
	})
}

// servicesFixtureRoot resolves the services fixture path.
func servicesFixtureRoot(t *testing.T) string {
	t.Helper()
	return fixtureRoot(t, "services")
}

// collectSinkBody concatenates every written entry from mem
// into one string for grep-style assertions.
func collectSinkBody(mem *sink.Memory) string {
	var b strings.Builder
	for k, v := range mem.Files() {
		b.WriteString("--- ")
		b.WriteString(k.Dir)
		b.WriteByte('/')
		b.WriteString(k.Filename)
		b.WriteString(" ---\n")
		b.Write(v)
		b.WriteByte('\n')
	}
	return b.String()
}
