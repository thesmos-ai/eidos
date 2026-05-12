// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package referenceacceptance_test

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/demopipe"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/auditweaver"
	"go.thesmos.sh/eidos/reference/buildergen"
	"go.thesmos.sh/eidos/reference/debugweaver"
	"go.thesmos.sh/eidos/reference/mockgen"
	"go.thesmos.sh/eidos/reference/registrygen"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/reference/shapewriter"
	"go.thesmos.sh/eidos/sink"
)

// outputPackage is the destination every emit-bound plugin routes
// its output through, so foundation + composition + cross-cutting
// contributions compose into one rendered file tree.
const outputPackage = "gen"

// TestEndToEnd drives every reference plugin against the
// demoproject fixture and asserts the cross-plugin acceptance
// criteria: clean diagnostics, every plugin's output reaches the
// sink, repogen-emitted methods carry both weaver contributions
// in the correct order, and the registry-gen init block lands
// inside one func init().
func TestEndToEnd(t *testing.T) {
	t.Parallel()

	result := runAllPlugins(t)
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}

	t.Run("repogen-emitted methods carry debug-then-audit prebody", func(t *testing.T) {
		t.Parallel()
		// Under the centralised layout each plugin composes
		// `<source-basename><plugin-suffix>` for its rendered file,
		// so repogen's ArticleRepository body lands in
		// `article<_repo.go>`. Both prebody contributions render
		// into that file in capability-topo order (debug then audit).
		body := sinkBodyFromResult(t, result, "article"+repogen.FilenameSuffix)
		debugIdx := strings.Index(body, `log.Printf("debug: %s entered", "ArticleRepo.Get")`)
		auditIdx := strings.Index(body, `audit.Record("%s", "ArticleRepo.Get")`)
		if debugIdx < 0 || auditIdx < 0 {
			t.Fatalf("rendered file missing prebody contributions: debug=%d audit=%d", debugIdx, auditIdx)
		}
		if debugIdx >= auditIdx {
			t.Fatalf("debug must render before audit; got debug=%d audit=%d", debugIdx, auditIdx)
		}
	})

	t.Run("buildergen + repogen each render into per-suffix files", func(t *testing.T) {
		t.Parallel()
		repo := sinkBodyFromResult(t, result, "article"+repogen.FilenameSuffix)
		bld := sinkBodyFromResult(t, result, "article"+buildergen.FilenameSuffix)
		if !strings.Contains(repo, "type ArticleRepository interface") {
			t.Fatalf("repo file missing ArticleRepository; got:\n%s", repo)
		}
		if !strings.Contains(bld, "type ArticleBuilder struct") {
			t.Fatalf("builder file missing ArticleBuilder; got:\n%s", bld)
		}
	})

	t.Run("mockgen produces a mock per emit interface plus the +gen:mock source interface", func(t *testing.T) {
		t.Parallel()
		// Mockgen composes its mock files from the upstream
		// interface's origin (Article struct → article_mock.go;
		// Searcher interface → searcher_mock.go).
		article := sinkBodyFromResult(t, result, "article"+mockgen.FilenameSuffix)
		searcher := sinkBodyFromResult(t, result, "searcher"+mockgen.FilenameSuffix)
		if !strings.Contains(article, "type ArticleRepositoryMock struct") {
			t.Fatalf("mock file missing ArticleRepositoryMock; got:\n%s", article)
		}
		if !strings.Contains(searcher, "type SearcherMock struct") {
			t.Fatalf("mock file missing SearcherMock; got:\n%s", searcher)
		}
	})

	t.Run("registry-gen produces one func init with the Article registration", func(t *testing.T) {
		t.Parallel()
		registry := sinkBodyFromResult(t, result, "registry.go")
		if want := `registry.Register("Article", blog.Article{})`; !strings.Contains(registry, want) {
			t.Fatalf("registry.go missing %q; got:\n%s", want, registry)
		}
		if strings.Count(registry, "func init()") != 1 {
			t.Fatalf("expected exactly one func init() block in registry.go; got:\n%s", registry)
		}
	})

	t.Run("shape-writer detects LineWriter and skips Article", func(t *testing.T) {
		t.Parallel()
		lw, ok := result.Store.Nodes().Structs().ByQName("example.com/demoproject/blog.LineWriter")
		if !ok {
			t.Fatalf("LineWriter missing from node store")
		}
		if detected, _ := shapewriter.Detected.Get(lw.Meta()); !detected {
			t.Fatalf("LineWriter should be detected as writer-shaped")
		}
		art, ok := result.Store.Nodes().Structs().ByQName("example.com/demoproject/blog.Article")
		if !ok {
			t.Fatalf("Article missing from node store")
		}
		if detected, _ := shapewriter.Detected.Get(art.Meta()); detected {
			t.Fatalf("Article should not be detected as writer-shaped")
		}
	})
}

// TestEndToEnd_ByteStable runs the full pipeline twice and asserts
// the rendered sink contents — and the per-file provenance hashes
// — match across runs. Determinism is the spec's headline quality
// contract; this test pins it for the multi-plugin path.
func TestEndToEnd_ByteStable(t *testing.T) {
	t.Parallel()

	first := snapshotFiles(t, runAllPlugins(t).Sink)
	second := snapshotFiles(t, runAllPlugins(t).Sink)
	if len(first) != len(second) {
		t.Fatalf("file count differs across runs: first=%d second=%d", len(first), len(second))
	}
	for path, body := range first {
		other, ok := second[path]
		if !ok {
			t.Fatalf("%s present in first run but missing in second", path)
		}
		if body != other {
			t.Fatalf("%s differs across runs:\n--- first ---\n%s\n--- second ---\n%s", path, body, other)
		}
	}
}

// runAllPlugins wires every reference plugin against the
// demoproject fixture with the centralised layout selected through
// the routing-layer surface so foundation + composition +
// cross-cutting contributions all share an output directory.
func runAllPlugins(t *testing.T) demopipe.Result {
	t.Helper()
	return demopipe.Run(t, demopipe.RunOptions{
		Annotators: []plugin.Annotator{shapewriter.New()},
		Generators: []plugin.Generator{
			repogen.New(),
			buildergen.New(),
			mockgen.New(),
			debugweaver.New(),
			auditweaver.New(),
			registrygen.New(),
		},
		Backend:       backend_golang.New(),
		Layout:        pipeline.LayoutCentralised,
		OutputPackage: outputPackage,
		PluginOptions: map[string]map[string]string{
			// registrygen synthesises an emit.File via FileFor at
			// generator time; the Layout phase does not re-route
			// pre-built Files because they carry no source Origin.
			// Until the plugin migrates to origin-anchored slot
			// attachment, its plugin-level output_package option
			// is the path that drives centralised routing.
			registrygen.Name: {
				"output_package":   outputPackage,
				"register_package": "registry",
				"register_func":    "Register",
			},
			auditweaver.Name: {
				"package": "audit",
				"func":    "Record",
				"format":  "%s",
			},
		},
	})
}

// snapshotFiles returns a deterministic map of file-path → SHA-256
// of body bytes for the captured sink. The hash isolates content
// from line-ending or comment-text noise without requiring the
// caller to reason about gofmt artefacts.
func snapshotFiles(t *testing.T, s sink.Sink) map[string]string {
	t.Helper()
	mem, ok := s.(*sink.Memory)
	if !ok {
		t.Fatalf("sink is not *sink.Memory; got %T", s)
	}
	out := make(map[string]string, mem.Len())
	for target, body := range mem.Files() {
		sum := sha256.Sum256(body)
		out[target.JoinPath()] = hex.EncodeToString(sum[:])
	}
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return out
}

// sinkBodyFromResult returns the rendered body of filename routed
// through outputPackage, surfaced from result's sink for downstream
// assertions.
func sinkBodyFromResult(t *testing.T, r demopipe.Result, filename string) string {
	t.Helper()
	mem, ok := r.Sink.(*sink.Memory)
	if !ok {
		t.Fatalf("sink is not *sink.Memory; got %T", r.Sink)
	}
	for target, body := range mem.Files() {
		if target.Filename == filename && target.Package == outputPackage {
			return string(body)
		}
	}
	keys := make([]string, 0)
	for k := range mem.Files() {
		keys = append(keys, k.JoinPath())
	}
	sort.Strings(keys)
	t.Fatalf("sink missing %q under package %q; available paths: %v", filename, outputPackage, keys)
	return ""
}

// _ keeps the emit import live for cross-package test-helper
// shapes used by future end-to-end assertions (Target value
// construction, ref shape comparisons). Drop once the helpers
// arrive.
var _ = emit.Target{}
