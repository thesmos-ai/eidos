// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protopipe_test

import (
	"bytes"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/protopipe"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/auditweaver"
	"go.thesmos.sh/eidos/reference/buildergen"
	"go.thesmos.sh/eidos/reference/debugweaver"
	"go.thesmos.sh/eidos/reference/mockgen"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/sink"
)

// TestRun_ServicesMockgen pins the directive-driven mockgen gate
// against a proto-derived interface: the protobuf frontend
// produces a node.Interface carrying `+gen:mock` on its
// DirectiveList; mockgen sees the source-side opt-in and emits
// one mock-struct emit entry per service. The rendered Go output
// isn't asserted here — that lands once the protogo bridge
// translates the proto-native type names — but the directive
// gate firing is the contract this test enforces.
func TestRun_ServicesMockgen(t *testing.T) {
	t.Parallel()

	t.Run("proto service carrying +gen:mock produces a mock struct emit entry", func(t *testing.T) {
		t.Parallel()
		root := servicesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir:  root,
			Generators: []plugin.Generator{mockgen.New()},
			Backend:    backend_golang.New(),
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		structs := result.Store.Emit().Structs().Items()
		var found *emit.Struct
		for _, s := range structs {
			if s.Name == "GreeterMock" {
				found = s
				break
			}
		}
		if found == nil {
			names := make([]string, 0, len(structs))
			for _, s := range structs {
				names = append(names, s.Name)
			}
			t.Fatalf("expected an emit struct named GreeterMock; got %+v", names)
		}
	})
}

// TestRun_ServicesMockgen_NegativeDirective pins the directive
// gate's negative path: a service carrying `-gen:mock` is opted
// out of mockgen even when the plugin is registered. Greeter
// (with `+gen:mock`) still produces its mock; SilentService
// (with `-gen:mock`) does not.
func TestRun_ServicesMockgen_NegativeDirective(t *testing.T) {
	t.Parallel()

	t.Run("-gen:mock on a service suppresses the mock-struct emit entry", func(t *testing.T) {
		t.Parallel()
		root := servicesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir:  root,
			Generators: []plugin.Generator{mockgen.New()},
			Backend:    backend_golang.New(),
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		structs := result.Store.Emit().Structs().Items()
		gotNames := make([]string, 0, len(structs))
		for _, s := range structs {
			gotNames = append(gotNames, s.Name)
		}
		var hasGreeter, hasSilent bool
		for _, n := range gotNames {
			if n == "GreeterMock" {
				hasGreeter = true
			}
			if n == "SilentServiceMock" {
				hasSilent = true
			}
		}
		if !hasGreeter {
			t.Fatalf("GreeterMock missing from emit structs; got %+v", gotNames)
		}
		if hasSilent {
			t.Fatalf("SilentServiceMock should be suppressed by -gen:mock; got %+v", gotNames)
		}
	})
}

// TestRun_ServicesMockgen_UniformAcrossStreaming pins mockgen's
// reference behaviour: emitted methods share one shape across
// unary and streaming RPCs. mockgen does not consult streaming
// meta — every RPC produces the same func-field-plus-dispatcher
// pattern. The test cross-checks the emit-method count and
// confirms each rendered method on the mock carries the same
// param + return arity, so a future regression toward streaming-
// aware emission fails here.
func TestRun_ServicesMockgen_UniformAcrossStreaming(t *testing.T) {
	t.Parallel()

	t.Run("mockgen emits one uniform method per RPC regardless of streaming flavour", func(t *testing.T) {
		t.Parallel()
		root := servicesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir:  root,
			Generators: []plugin.Generator{mockgen.New()},
			Backend:    backend_golang.New(),
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		methods := result.Store.Emit().Methods().Items()
		greeterMethods := make([]*emit.Method, 0, len(methods))
		for _, m := range methods {
			owner, ok := m.Owner.(*emit.Struct)
			if !ok || owner.Name != "GreeterMock" {
				continue
			}
			greeterMethods = append(greeterMethods, m)
		}
		// CollectNames carries its own -gen:mock, so it is skipped
		// per the per-RPC opt-out path. The remaining four RPCs
		// (one unary, one server-stream, one bidi, one cross-package
		// unary) all surface as mock methods.
		want := []string{"SayHello", "SayHellos", "Chat", "Echo"}
		gotNames := make([]string, 0, len(greeterMethods))
		for _, m := range greeterMethods {
			gotNames = append(gotNames, m.Name)
		}
		sort.Strings(gotNames)
		sortedWant := append([]string{}, want...)
		sort.Strings(sortedWant)
		if !equalStringSlicesOrdered(gotNames, sortedWant) {
			t.Fatalf("GreeterMock methods = %+v, want %+v (unordered)", gotNames, sortedWant)
		}
		// Uniformity check: every emit method on the mock has one
		// param and one return value — the unary shape mockgen
		// applies uniformly. A future streaming-aware mockgen would
		// emit different arities for streaming flavours; this
		// assertion locks in the documented reference behaviour.
		for _, m := range greeterMethods {
			if got := len(m.Params); got != 1 {
				t.Fatalf("GreeterMock.%s Params has %d entries, want 1", m.Name, got)
			}
			if got := len(m.Returns); got != 1 {
				t.Fatalf("GreeterMock.%s Returns has %d entries, want 1", m.Name, got)
			}
		}
	})
}

// TestRun_ServicesMockgen_RenderByteStable pins the documented
// determinism contract on the rendered Go output: two consecutive
// runs against the same fixture produce byte-identical sink
// contents. A regression in iteration order, meta-key sequence,
// or template rendering would diverge the bytes and fail this
// assertion.
func TestRun_ServicesMockgen_RenderByteStable(t *testing.T) {
	t.Parallel()

	t.Run("two consecutive runs produce byte-identical sink contents", func(t *testing.T) {
		t.Parallel()
		root := servicesFixtureRoot(t)
		first := renderServicesFixture(t, root)
		second := renderServicesFixture(t, root)
		if !bytes.Equal(first, second) {
			t.Fatalf(
				"rendered output diverged across two runs:\n--- first ---\n%s\n--- second ---\n%s",
				first, second,
			)
		}
	})
}

// TestRun_ServicesGeneratorScoping pins the framework's directive-
// kind scoping against proto-derived services: registering
// repogen and buildergen alongside mockgen produces no emit
// entries from those plugins because the proto frontend doesn't
// produce KindStruct hosts for services. The assertion guards
// against accidental scope-creep should the directive system's
// kind-filter ever break.
func TestRun_ServicesGeneratorScoping(t *testing.T) {
	t.Parallel()

	t.Run("repogen and buildergen produce no entries for proto services", func(t *testing.T) {
		t.Parallel()
		root := servicesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir: root,
			Generators: []plugin.Generator{
				mockgen.New(),
				buildergen.New(),
				repogen.New(),
			},
			Backend: backend_golang.New(),
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		// Greeter / SilentService should never produce a Builder
		// or Repository struct — those plugins target KindStruct
		// hosts, and the proto frontend's services are KindInterface.
		for _, s := range result.Store.Emit().Structs().Items() {
			switch s.Name {
			case "GreeterBuilder", "SilentServiceBuilder",
				"GreeterRepository", "SilentServiceRepository":
				t.Fatalf(
					"service-scoped emit entity %q should not be produced; got it from a generator that targets KindStruct only",
					s.Name,
				)
			}
		}
	})
}

// renderServicesFixture runs the protobuf frontend + mockgen +
// Go backend pipeline against the services fixture and returns
// the bytes the sink captured. Used by the byte-stability test
// to compare two consecutive runs.
func renderServicesFixture(t *testing.T, root string) []byte {
	t.Helper()
	mem := sink.NewMemory()
	result := protopipe.Run(t, protopipe.RunOptions{
		SourceDir:  root,
		Generators: []plugin.Generator{mockgen.New()},
		Backend:    backend_golang.New(),
		Sink:       mem,
	})
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	files := mem.Files()
	keys := make([]emit.Target, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Dir != keys[j].Dir {
			return keys[i].Dir < keys[j].Dir
		}
		return keys[i].Filename < keys[j].Filename
	})
	var buf bytes.Buffer
	for _, k := range keys {
		buf.WriteString("--- ")
		buf.WriteString(k.Dir)
		buf.WriteString("/")
		buf.WriteString(k.Filename)
		buf.WriteString(" ---\n")
		buf.Write(files[k])
		buf.WriteString("\n")
	}
	return buf.Bytes()
}

// equalStringSlicesOrdered reports whether got and want are
// element-for-element equal.
func equalStringSlicesOrdered(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestRun_ServicesMockgen_PerMethodNegativeDirective pins the
// per-RPC opt-out path: a `-gen:mock` directive on an individual
// rpc skips that single method while the surrounding service's
// other RPCs continue to mock normally.
func TestRun_ServicesMockgen_PerMethodNegativeDirective(t *testing.T) {
	t.Parallel()

	t.Run("-gen:mock on a single RPC skips that method without affecting the rest of the service", func(t *testing.T) {
		t.Parallel()
		root := servicesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir:  root,
			Generators: []plugin.Generator{mockgen.New()},
			Backend:    backend_golang.New(),
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		methods := result.Store.Emit().Methods().Items()
		gotNames := make([]string, 0, len(methods))
		for _, m := range methods {
			owner, ok := m.Owner.(*emit.Struct)
			if !ok || owner.Name != "GreeterMock" {
				continue
			}
			gotNames = append(gotNames, m.Name)
		}
		sort.Strings(gotNames)
		wantNames := []string{"Chat", "Echo", "SayHello", "SayHellos"}
		if !equalStringSlicesOrdered(gotNames, wantNames) {
			t.Fatalf("GreeterMock methods = %+v, want %+v (CollectNames suppressed)", gotNames, wantNames)
		}
	})
}

// TestRun_ServicesMockgenWithWeavers pins the cross-cutting
// weaver attachment against proto-derived RPCs: registering
// debugweaver and auditweaver alongside mockgen attaches a
// prebody contribution from each weaver onto every emit method
// mockgen produced. The contract holds for unary and streaming
// RPCs uniformly because mockgen's emitted methods share one
// shape regardless of streaming flavour.
func TestRun_ServicesMockgenWithWeavers(t *testing.T) {
	t.Parallel()

	t.Run("debug- and audit-weavers each contribute a prebody on every mock method", func(t *testing.T) {
		t.Parallel()
		root := servicesFixtureRoot(t)
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir: root,
			Generators: []plugin.Generator{
				mockgen.New(),
				debugweaver.New(),
				auditweaver.New(),
			},
			Backend: backend_golang.New(),
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		methods := result.Store.Emit().Methods().Items()
		mockMethods := make([]*emit.Method, 0, len(methods))
		for _, m := range methods {
			if owner, ok := m.Owner.(*emit.Struct); ok && owner.Name == "GreeterMock" {
				mockMethods = append(mockMethods, m)
			}
		}
		if len(mockMethods) == 0 {
			t.Fatalf("expected GreeterMock to carry emit methods; none found")
		}
		for _, m := range mockMethods {
			items := m.Prebody().Items
			if len(items) < 2 {
				t.Fatalf(
					"GreeterMock.%s prebody has %d items, want at least 2 (debug + audit)",
					m.Name, len(items),
				)
			}
		}
	})
}

// servicesFixtureRoot resolves the absolute path of the
// frontend/protobuf/testdata/services directory. The protopipe
// harness defaults to its own testdata tree, so a test that
// drives mockgen against the frontend's services fixture has to
// point at the frontend's testdata root explicitly.
func servicesFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	return filepath.Join(repoRoot, "frontend", "protobuf", "testdata", "services")
}
