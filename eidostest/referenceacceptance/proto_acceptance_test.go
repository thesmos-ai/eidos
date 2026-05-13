// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package referenceacceptance_test

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/bridge/protogo"
	"go.thesmos.sh/eidos/eidostest/protopipe"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/auditweaver"
	"go.thesmos.sh/eidos/reference/buildergen"
	"go.thesmos.sh/eidos/reference/debugweaver"
	"go.thesmos.sh/eidos/reference/mockgen"
	"go.thesmos.sh/eidos/reference/registrygen"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/sink"
)

// protoOutputPackage is the centralised destination every proto-
// acceptance plugin routes through, so foundation + composition +
// cross-cutting contributions compose into one rendered file
// tree just like the Go-input parallel under [TestEndToEnd].
const protoOutputPackage = "gen"

// TestEndToEnd_Proto drives every reference plugin against the
// proto buildfixture and asserts the cross-plugin contract the
// Go-input parallel pins: clean diagnostics; buildergen + repogen
// render into per-suffix files; mockgen surfaces both the
// directly-annotated source service and the repogen-emitted
// repository interface; registrygen produces one `func init()`
// carrying the User registration; the cross-cutting weavers'
// prebody contributions land in capability-topo order on
// repogen-emitted methods.
func TestEndToEnd_Proto(t *testing.T) {
	t.Parallel()

	result := runAllProtoPlugins(t)
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}

	t.Run("repogen-emitted methods carry debug-then-audit prebody", func(t *testing.T) {
		t.Parallel()
		body := sinkBodyFromProtoResult(t, result, "buildfixture"+repogen.FilenameSuffix)
		debugIdx := strings.Index(body, `log.Printf("debug: %s entered", "UserRepo.Get")`)
		auditIdx := strings.Index(body, `audit.Record("%s", "UserRepo.Get")`)
		if debugIdx < 0 || auditIdx < 0 {
			t.Fatalf("rendered file missing prebody contributions: debug=%d audit=%d\n%s",
				debugIdx, auditIdx, body)
		}
		if debugIdx >= auditIdx {
			t.Fatalf("debug must render before audit; got debug=%d audit=%d", debugIdx, auditIdx)
		}
	})

	t.Run("buildergen + repogen each render into per-suffix files", func(t *testing.T) {
		t.Parallel()
		bld := sinkBodyFromProtoResult(t, result, "buildfixture"+buildergen.FilenameSuffix)
		repo := sinkBodyFromProtoResult(t, result, "buildfixture"+repogen.FilenameSuffix)
		if !strings.Contains(bld, "type UserBuilder struct") {
			t.Fatalf("builder file missing UserBuilder; got:\n%s", bld)
		}
		if !strings.Contains(repo, "type UserRepository interface") {
			t.Fatalf("repo file missing UserRepository; got:\n%s", repo)
		}
	})

	t.Run("mockgen mocks the directive-annotated source service and the repogen-emitted interface", func(t *testing.T) {
		t.Parallel()
		body := sinkBodyFromProtoResult(t, result, "buildfixture"+mockgen.FilenameSuffix)
		if !strings.Contains(body, "type UserServiceMock struct") {
			t.Fatalf("mock file missing UserServiceMock; got:\n%s", body)
		}
		if !strings.Contains(body, "type UserRepositoryMock struct") {
			t.Fatalf("mock file missing UserRepositoryMock; got:\n%s", body)
		}
	})

	t.Run("registry-gen produces one func init with the User registration", func(t *testing.T) {
		t.Parallel()
		body := sinkBodyFromProtoResult(t, result, "buildfixture"+registrygen.FilenameSuffix)
		if want := `registry.Register("User", buildfixture.User{})`; !strings.Contains(body, want) {
			t.Fatalf("registry file missing %q; got:\n%s", want, body)
		}
		if strings.Count(body, "func init()") != 1 {
			t.Fatalf("expected exactly one func init() block; got:\n%s", body)
		}
	})
}

// TestEndToEnd_ProtoByteStable runs the full proto pipeline twice
// and asserts the rendered sink contents and per-file provenance
// hashes match across runs. Determinism is the spec's headline
// contract; this test pins it for the multi-plugin proto path so
// any future non-determinism (map iteration, fresh allocation
// order, ...) surfaces here.
func TestEndToEnd_ProtoByteStable(t *testing.T) {
	t.Parallel()

	first := snapshotProtoFiles(t, runAllProtoPlugins(t).Sink)
	second := snapshotProtoFiles(t, runAllProtoPlugins(t).Sink)
	if len(first) != len(second) {
		t.Fatalf("file count differs across runs: first=%d second=%d", len(first), len(second))
	}
	for path, body := range first {
		other, ok := second[path]
		if !ok {
			t.Fatalf("%s present in first run but missing in second", path)
		}
		if body != other {
			t.Fatalf("%s differs across runs:\n--- first ---\n%s\n--- second ---\n%s",
				path, body, other)
		}
	}
}

// runAllProtoPlugins wires every reference plugin against the
// proto buildfixture with the centralised layout selected through
// the routing-layer surface so foundation + composition +
// cross-cutting contributions all share an output directory.
//
// The pinned [protopipe.RunOptions.Command] keeps the rendered
// `Command:` header byte-identical across `go test` invocations,
// so the byte-stable assertion is independent of the test
// binary's per-machine invocation arguments.
func runAllProtoPlugins(t *testing.T) protopipe.Result {
	t.Helper()
	return protopipe.Run(t, protopipe.RunOptions{
		SourceDir:  protoFixtureRoot(t),
		Annotators: []plugin.Annotator{protogo.New()},
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
		OutputPackage: protoOutputPackage,
		Command:       "go test (referenceacceptance, proto)",
		PluginOptions: map[string]map[string]string{
			registrygen.Name: {
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

// protoFixtureRoot resolves the absolute path of the
// eidostest/protopipe/buildfixture directory through
// [runtime.Caller], so the resolved path is stable regardless of
// the test's working directory.
func protoFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	return filepath.Join(dir, "..", "protopipe", "buildfixture")
}

// snapshotProtoFiles returns a deterministic map of file-path →
// SHA-256 of body bytes for the captured sink. Mirrors the
// snapshotFiles helper used by the Go-input baseline; isolated
// here so the proto byte-stable test stays self-contained.
func snapshotProtoFiles(t *testing.T, s sink.Sink) map[string]string {
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

// sinkBodyFromProtoResult returns the rendered body of filename
// routed through protoOutputPackage, surfaced from r's sink for
// downstream substring assertions.
func sinkBodyFromProtoResult(t *testing.T, r protopipe.Result, filename string) string {
	t.Helper()
	mem, ok := r.Sink.(*sink.Memory)
	if !ok {
		t.Fatalf("sink is not *sink.Memory; got %T", r.Sink)
	}
	for target, body := range mem.Files() {
		if target.Filename == filename && target.Package == protoOutputPackage {
			return string(body)
		}
	}
	keys := make([]string, 0)
	for k := range mem.Files() {
		keys = append(keys, k.JoinPath())
	}
	sort.Strings(keys)
	t.Fatalf("sink missing %q under package %q; available paths: %v",
		filename, protoOutputPackage, keys)
	return ""
}
