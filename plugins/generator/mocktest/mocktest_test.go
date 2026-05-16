// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mocktest_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/plugins/generator/mock"
	"go.thesmos.sh/eidos/plugins/generator/mocktest"
	"go.thesmos.sh/eidos/priority"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/store"
)

// TestPlugin_Contract pins the framework contract surface — name,
// priority, capability declarations, role implementation, and
// empty-store no-op behaviour.
func TestPlugin_Contract(t *testing.T) {
	t.Parallel()

	t.Run("Name matches the package constant", func(t *testing.T) {
		t.Parallel()
		if got := mocktest.New().Name(); got != mocktest.Name {
			t.Fatalf("Name() = %q, want %q", got, mocktest.Name)
		}
	})

	t.Run("Priority is GeneratorCrossCutting", func(t *testing.T) {
		t.Parallel()
		if got, want := mocktest.New().Priority(), priority.GeneratorCrossCutting; got != want {
			t.Fatalf("Priority() = %v, want %v", got, want)
		}
	})

	t.Run("Requires the mock capability", func(t *testing.T) {
		t.Parallel()
		got := mocktest.New().Requires()
		if len(got) != 1 || got[0] != mock.Capability {
			t.Fatalf("Requires() = %v, want [%q]", got, mock.Capability)
		}
	})

	t.Run("satisfies plugin.Generator", func(t *testing.T) {
		t.Parallel()
		var _ plugin.Generator = mocktest.New()
	})

	t.Run("Generate on empty emit store is a no-op", func(t *testing.T) {
		t.Parallel()
		ctx := newGeneratorContext(t, store.New())
		if err := mocktest.New().Generate(ctx); err != nil {
			t.Fatalf("Generate(empty): %v", err)
		}
	})
}

// TestPlugin_EmitsTests covers the canonical happy path — one
// mock struct yields one Test<MockName> function whose body nests
// per-method subtests, each carrying two branch subtests (nil +
// override dispatch). This shape lets `go test -run` filter by
// method (`TestStoreMock/Get`) or branch
// (`TestStoreMock/Get/nil dispatch`) without bespoke flag plumbing.
func TestPlugin_EmitsTests(t *testing.T) {
	t.Parallel()

	ctx := runMockThenMockTest(t, oneMethodInterface())

	got := findFunction(ctx.Store.Emit(), "x.TestStoreMock")
	if got == nil {
		t.Fatalf("test function TestStoreMock not emitted; "+
			"got %d functions in emit", ctx.Store.Emit().Functions().Len())
	}

	t.Run("body nests method name as a subtest", func(t *testing.T) {
		t.Parallel()
		rendered := renderStmts(got.Body)
		if !strings.Contains(rendered, "Get") {
			t.Fatalf("body missing Get method subtest; got:\n%s", rendered)
		}
	})

	t.Run("body carries both branch subtests for 100% coverage", func(t *testing.T) {
		t.Parallel()
		rendered := renderStmts(got.Body)
		if !strings.Contains(rendered, "nil dispatch") {
			t.Fatalf("body missing nil-dispatch subtest; got:\n%s", rendered)
		}
		if !strings.Contains(rendered, "override dispatch") {
			t.Fatalf("body missing override-dispatch subtest; got:\n%s", rendered)
		}
	})

	t.Run("two-method interface yields one function with two method subtests", func(t *testing.T) {
		t.Parallel()
		ctx := runMockThenMockTest(t, twoMethodInterface())
		got := findFunction(ctx.Store.Emit(), "x.TestStoreMock")
		if got == nil {
			t.Fatalf("TestStoreMock not emitted")
		}
		rendered := renderStmts(got.Body)
		for _, want := range []string{"Get", "Put"} {
			if !strings.Contains(rendered, want) {
				t.Fatalf("body missing %s method subtest; got:\n%s", want, rendered)
			}
		}
	})
}

// TestPlugin_RespectsOptions covers the configurable surface —
// custom filename suffix and test package routing.
func TestPlugin_RespectsOptions(t *testing.T) {
	t.Parallel()

	t.Run("FilenameSuffix returns the configured suffix", func(t *testing.T) {
		t.Parallel()
		p := mocktest.New()
		if got := p.FilenameSuffix("golang"); got != "_mock_test.go" {
			t.Fatalf("FilenameSuffix(golang) = %q, want %q", got, "_mock_test.go")
		}
		if got := p.FilenameSuffix("rust"); got != "" {
			t.Fatalf("FilenameSuffix(rust) = %q, want empty", got)
		}
	})

	t.Run("custom Suffix option flows into emitted filename suffix", func(t *testing.T) {
		t.Parallel()
		p := mocktest.New()
		opts := opt.New(p.OptionsSchema(), map[string]string{
			"test_filename_suffix": "_mymock_test.go",
		})
		if err := p.SetOptions(opts); err != nil {
			t.Fatalf("SetOptions: %v", err)
		}
		if got := p.FilenameSuffix("golang"); got != "_mymock_test.go" {
			t.Fatalf("FilenameSuffix(golang) = %q, want %q", got, "_mymock_test.go")
		}
	})
}

// TestPlugin_SkipsNonMockStructs covers the meta-key gate —
// emit structs without the mock.iface stamp are ignored.
func TestPlugin_SkipsNonMockStructs(t *testing.T) {
	t.Parallel()

	s := store.New()
	// Add an emit struct without the mock.iface stamp.
	pkg := &emit.Package{Name: "x", Path: "x"}
	pkg.Structs = append(pkg.Structs, &emit.Struct{
		BaseEmit: emit.BaseEmit{},
		Name:     "JustAStruct",
		Package:  "x",
	})
	if err := s.Emit().AddPackage(pkg); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}

	ctx := newGeneratorContext(t, s)
	if err := mocktest.New().Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got := s.Emit().Functions().Len(); got != 0 {
		t.Fatalf("expected no test functions for non-mock struct; got %d", got)
	}
}

// oneMethodInterface returns a fixture node.Interface with one
// method (Get) annotated with +gen:mock — the canonical single-
// method input for mock + mocktest round-trip tests.
func oneMethodInterface() *node.Interface {
	return &node.Interface{
		Name: "Store", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{{Name: mock.DirectiveName}},
		},
		Methods: []*node.Method{
			{
				Name: "Get",
				Params: []*node.Param{
					{Name: "ctx", Type: &node.TypeRef{Name: "Context", Package: "context"}},
					{Name: "key", Type: &node.TypeRef{Name: "string"}},
				},
				Returns: []*node.TypeRef{
					{Name: "Record", Package: "x"},
					{Name: "error"},
				},
			},
		},
	}
}

// twoMethodInterface adds a second method (Put) so per-method
// emit-count assertions have something to fan out across.
func twoMethodInterface() *node.Interface {
	i := oneMethodInterface()
	i.Methods = append(i.Methods, &node.Method{
		Name: "Put",
		Params: []*node.Param{
			{Name: "ctx", Type: &node.TypeRef{Name: "Context", Package: "context"}},
			{Name: "r", Type: &node.TypeRef{Name: "Record", Package: "x"}},
		},
		Returns: []*node.TypeRef{{Name: "error"}},
	})
	return i
}

// runMockThenMockTest wires iface into a fresh store, runs the
// mock plugin to populate the emit-side mock struct, then runs
// the mocktest plugin against the resulting emit graph. Returns
// the context whose store now contains both emit outputs.
func runMockThenMockTest(t *testing.T, iface *node.Interface) *sdk.GeneratorContext {
	t.Helper()
	s := store.New()
	if err := s.Nodes().AddPackage(&node.Package{
		Name: "x", Path: "x",
		Interfaces: []*node.Interface{iface},
	}); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}

	mockCtx := newGeneratorContext(t, s)
	if err := mock.New().Generate(mockCtx); err != nil {
		t.Fatalf("mock.Generate: %v", err)
	}

	testCtx := newGeneratorContext(t, s)
	if err := mocktest.New().Generate(testCtx); err != nil {
		t.Fatalf("mocktest.Generate: %v", err)
	}
	return testCtx
}

// newGeneratorContext wraps s in a fresh [sdk.GeneratorContext]
// with read-tracking reader and an empty diagnostic sink.
func newGeneratorContext(t *testing.T, s *store.Store) *sdk.GeneratorContext {
	t.Helper()
	return &sdk.GeneratorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   diag.New(),
	}
}

// findFunction returns the emit function registered under qname,
// or nil when no such function exists.
func findFunction(v *store.EmitView, qname string) *emit.Function {
	got, _ := v.Functions().ByQName(qname)
	return got
}

// renderStmts dumps the body as a flat string so assertions can
// search for subtest names without descending into the
// statement tree. The dump form is intentionally crude — only
// the identifier and string-literal text are surfaced.
func renderStmts(body []*emit.Stmt) string {
	var b strings.Builder
	for _, s := range body {
		renderStmt(&b, s)
	}
	return b.String()
}

// renderStmt walks one statement, surfacing every contained
// identifier name + literal text into b. Sufficient for the
// "does this body mention X" assertion the tests run.
func renderStmt(b *strings.Builder, s *emit.Stmt) {
	if s == nil {
		return
	}
	renderExpr(b, s.Cond)
	renderExpr(b, s.RangeOver)
	renderExpr(b, s.Call)
	for _, e := range s.Targets {
		renderExpr(b, e)
	}
	for _, e := range s.Values {
		renderExpr(b, e)
	}
	for _, e := range s.Returns {
		renderExpr(b, e)
	}
	renderStmt(b, s.Init)
	renderStmt(b, s.Post)
	renderStmt(b, s.Inner)
	for _, c := range s.Block {
		renderStmt(b, c)
	}
	for _, c := range s.Else {
		renderStmt(b, c)
	}
	for _, c := range s.Cases {
		renderStmt(b, c)
	}
}

// renderExpr walks one expression, surfacing names + literal text.
func renderExpr(b *strings.Builder, e *emit.Expr) {
	if e == nil {
		return
	}
	if e.Name != "" {
		b.WriteString(e.Name)
		b.WriteByte(' ')
	}
	if e.RawText != "" {
		b.WriteString(e.RawText)
		b.WriteByte(' ')
	}
	renderExpr(b, e.Receiver)
	renderExpr(b, e.Callee)
	renderExpr(b, e.Left)
	renderExpr(b, e.Right)
	renderExpr(b, e.IndexExpr)
	renderExpr(b, e.Low)
	renderExpr(b, e.High)
	renderExpr(b, e.Max)
	for _, a := range e.Args {
		renderExpr(b, a)
	}
	for _, c := range e.FuncBody {
		renderStmt(b, c)
	}
}
