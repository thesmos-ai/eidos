// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mockgen_test

import (
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/demopipe"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/mockgen"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// outputPackage is the canonical destination package the demopipe
// harness pins via the routing-layer surface for the centralised
// end-to-end run.
const outputPackage = "gen"

// TestPluginShape pins the plugin's public-contract surface so a
// rename / drop accident surfaces at PR review time.
func TestPluginShape(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented identifier", func(t *testing.T) {
		t.Parallel()
		if got := mockgen.New().Name(); got != mockgen.Name {
			t.Fatalf("Name = %q, want %q", got, mockgen.Name)
		}
	})

	t.Run(
		"implements Generator, CapabilityProvider, OptionsProvider, DirectiveProvider, FilenameProvider",
		func(t *testing.T) {
			t.Parallel()
			p := mockgen.New()
			if _, ok := any(p).(plugin.Generator); !ok {
				t.Fatalf("plugin must implement plugin.Generator")
			}
			if _, ok := any(p).(plugin.CapabilityProvider); !ok {
				t.Fatalf("plugin must implement plugin.CapabilityProvider")
			}
			if _, ok := any(p).(plugin.OptionsProvider); !ok {
				t.Fatalf("plugin must implement plugin.OptionsProvider")
			}
			if _, ok := any(p).(plugin.DirectiveProvider); !ok {
				t.Fatalf("plugin must implement plugin.DirectiveProvider")
			}
			if _, ok := any(p).(plugin.FilenameProvider); !ok {
				t.Fatalf("plugin must implement plugin.FilenameProvider")
			}
		},
	)

	t.Run("Provides advertises the mock capability", func(t *testing.T) {
		t.Parallel()
		got := mockgen.New().Provides()
		if len(got) != 1 || got[0] != mockgen.Capability {
			t.Fatalf("Provides = %+v, want [%q]", got, mockgen.Capability)
		}
	})

	t.Run("Requires the repository capability", func(t *testing.T) {
		t.Parallel()
		got := mockgen.New().Requires()
		if len(got) != 1 || got[0] != mockgen.RequiresRepository {
			t.Fatalf("Requires = %+v, want [%q]", got, mockgen.RequiresRepository)
		}
	})

	t.Run("Directives returns the mock schema", func(t *testing.T) {
		t.Parallel()
		schemas := mockgen.New().Directives()
		if len(schemas) != 1 {
			t.Fatalf("expected one schema; got %d", len(schemas))
		}
		if schemas[0].Name != mockgen.DirectiveName {
			t.Fatalf("schema name = %q, want %q", schemas[0].Name, mockgen.DirectiveName)
		}
		if !schemas[0].AllowNegated {
			t.Fatalf("schema must allow the negated form for opt-out support")
		}
	})
}

// TestFilenameSuffix pins the language-parametric capability:
// only the supported backend language receives a suffix; other
// languages get the empty signal so the Layout phase can surface
// a missing-FilenameProvider error rather than producing a
// Go-shaped suffix for a non-Go renderer.
func TestFilenameSuffix(t *testing.T) {
	t.Parallel()

	t.Run("golang returns the test-file suffix", func(t *testing.T) {
		t.Parallel()
		p := mockgen.New()
		if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
			t.Fatalf("SetOptions: %v", err)
		}
		if got, want := p.FilenameSuffix(mockgen.Language), mockgen.FilenameSuffix; got != want {
			t.Fatalf("FilenameSuffix(%q) = %q, want %q", mockgen.Language, got, want)
		}
	})

	t.Run("unsupported language returns the empty signal", func(t *testing.T) {
		t.Parallel()
		p := mockgen.New()
		if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
			t.Fatalf("SetOptions: %v", err)
		}
		if got := p.FilenameSuffix("rust"); got != "" {
			t.Fatalf("FilenameSuffix(unsupported) = %q, want empty", got)
		}
	})
}

// TestGenerate_EndToEnd runs mockgen alongside repogen against the
// demoproject fixture and asserts the produced mock surface covers
// every targeted interface: repogen-emitted repositories plus the
// `+gen:mock` source interface.
func TestGenerate_EndToEnd(t *testing.T) {
	t.Parallel()

	result := demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{repogen.New(), mockgen.New()},
		Backend:       backend_golang.New(),
		Layout:        pipeline.LayoutCentralised,
		OutputPackage: outputPackage,
	})
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}

	t.Run("repogen-emitted repositories each get a mock counterpart", func(t *testing.T) {
		t.Parallel()
		for _, want := range []string{"ArticleRepositoryMock", "UserRepositoryMock"} {
			if !emitContainsStruct(result.Store, want) {
				t.Fatalf("expected emit store to contain %q", want)
			}
		}
	})

	t.Run("source +gen:mock interface produces a mock", func(t *testing.T) {
		t.Parallel()
		if !emitContainsStruct(result.Store, "SearcherMock") {
			t.Fatalf("expected emit store to contain SearcherMock (fixture's +gen:mock interface)")
		}
	})

	t.Run("rendered Searcher mock carries Func fields plus dispatch methods", func(t *testing.T) {
		t.Parallel()
		body := sinkBody(t, result.Sink, "searcher"+mockgen.FilenameSuffix)
		// Go's func-type rendering omits parameter names, so the
		// field types appear as `func(context.Context, string)` —
		// the dispatching method signatures preserve names. Both
		// are pinned below.
		for _, want := range []string{
			"type SearcherMock struct",
			"func(context.Context, string) (*blog.Article, error)",
			"func(context.Context, string) ([]*blog.Article, error)",
			"func (m *SearcherMock) Find(ctx context.Context, id string) (*blog.Article, error)",
			"return m.FindFunc(ctx, id)",
			"func (m *SearcherMock) Query(ctx context.Context, q string) ([]*blog.Article, error)",
			"return m.QueryFunc(ctx, q)",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("searcher.go missing %q; got:\n%s", want, body)
			}
		}
	})

	t.Run(
		"rendered ArticleRepository mock dispatches the canonical CRUD methods",
		func(t *testing.T) {
			t.Parallel()
			// Article mock composes into article<_mock.go> because its
			// Origin is the Article source struct (inherited from
			// repogen's ArticleRepository.Origin). The Layout phase
			// derives the source basename from that origin.
			body := sinkBody(t, result.Sink, "article"+mockgen.FilenameSuffix)
			for _, want := range []string{
				"type ArticleRepositoryMock struct",
				"func(context.Context, string) (*blog.Article, error)",
				"func (m *ArticleRepositoryMock) Get(ctx context.Context, id string) (*blog.Article, error)",
				"return m.GetFunc(ctx, id)",
				"func (m *ArticleRepositoryMock) List(ctx context.Context) ([]*blog.Article, error)",
				"return m.ListFunc(ctx)",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("article.go missing %q; got:\n%s", want, body)
				}
			}
		},
	)

	t.Run("Plan orders repogen before mockgen by priority bucket", func(t *testing.T) {
		t.Parallel()
		plan := planFor(t)
		repoIdx := indexOf(plan, repogen.Name)
		mockIdx := indexOf(plan, mockgen.Name)
		if repoIdx < 0 || mockIdx < 0 {
			t.Fatalf(
				"plan must contain both plugins; got repogen=%d mockgen=%d in %+v",
				repoIdx,
				mockIdx,
				plan,
			)
		}
		if repoIdx >= mockIdx {
			t.Fatalf("repogen must run before mockgen; got repogen=%d mockgen=%d", repoIdx, mockIdx)
		}
	})
}

// TestGenerate_HonoursScope pins the routing-layer scope contract.
// Both the source-side and emit-side mock paths iterate through
// scoped reader queries; under `-target Article` the source
// Searcher interface (no match) drops out, and the emit-side
// ArticleRepository (whose Origin is the Article source struct,
// in scope) still produces its mock. The Article source struct
// itself is the unit driving the scope predicate via its
// emit-side interface lineage.
func TestGenerate_HonoursScope(t *testing.T) {
	t.Parallel()

	result := demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{repogen.New(), mockgen.New()},
		Backend:       backend_golang.New(),
		Layout:        pipeline.LayoutCentralised,
		OutputPackage: outputPackage,
		TargetSymbol:  "Article",
	})
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}

	t.Run("ArticleRepositoryMock is in scope via its source-struct origin", func(t *testing.T) {
		t.Parallel()
		if !emitContainsStruct(result.Store, "ArticleRepositoryMock") {
			t.Fatalf("expected ArticleRepositoryMock under -target Article")
		}
	})

	t.Run("UserRepositoryMock and SearcherMock are out of scope", func(t *testing.T) {
		t.Parallel()
		for _, unwanted := range []string{"UserRepositoryMock", "SearcherMock"} {
			if emitContainsStruct(result.Store, unwanted) {
				t.Fatalf("did not expect %q under -target Article scope", unwanted)
			}
		}
	})
}

// TestGenerate_LeavesTargetForLayout pins the routing-layer
// contract: every emitted mock carries its source-anchored Origin
// but no plugin-stamped Target. The framework's Layout phase
// composes Target.Dir / Filename / Package / ImportPath
// downstream — the plugin never constructs an [emit.Target]
// literal.
func TestGenerate_LeavesTargetForLayout(t *testing.T) {
	t.Parallel()

	t.Run(
		"source-side mock anchors to the source interface and lands in the <srcPkg>_test emit.Package",
		func(t *testing.T) {
			t.Parallel()
			s := store.New()
			srcPkg := &node.Package{Name: "search", Path: "example.com/search"}
			iface := &node.Interface{
				Name:    "Finder",
				Package: srcPkg.Path,
				BaseNode: node.BaseNode{
					SourcePos:     position.Pos{File: "search/finder.go", Line: 1},
					DirectiveList: []*directive.Directive{{Name: mockgen.DirectiveName}},
				},
			}
			srcPkg.Interfaces = append(srcPkg.Interfaces, iface)
			if err := s.Nodes().AddPackage(srcPkg); err != nil {
				t.Fatalf("AddPackage: %v", err)
			}

			p := configuredPlugin(t)
			ctx := &plugin.GeneratorContext{
				Store: s, Reader: store.NewReader(s), Diag: diag.New(),
			}
			if err := p.Generate(ctx); err != nil {
				t.Fatalf("Generate: %v", err)
			}

			wantPath := srcPkg.Path + mockgen.TestPackageSuffix
			emitPkg, ok := s.Emit().Packages().ByQName(wantPath)
			if !ok {
				t.Fatalf(
					"expected emit.Package keyed by %q; got %+v",
					wantPath,
					s.Emit().Packages().Items(),
				)
			}
			if want := srcPkg.Name + mockgen.TestPackageSuffix; emitPkg.Name != want {
				t.Fatalf("emit.Package.Name = %q, want %q", emitPkg.Name, want)
			}
			if len(emitPkg.Structs) != 1 {
				t.Fatalf("expected one mock struct; got %d", len(emitPkg.Structs))
			}
			mock := emitPkg.Structs[0]
			if mock.Target != (emit.Target{}) {
				t.Fatalf("Target should be zero until Layout composes it; got %+v", mock.Target)
			}
			if mock.Origin() != iface {
				t.Fatalf("Origin should be the source interface so Layout can resolve the Target")
			}
		},
	)

	t.Run(
		"emit-side mock anchors to the upstream interface's Origin and lands in the <upstream>_test emit.Package",
		func(t *testing.T) {
			t.Parallel()
			s := store.New()
			// Stand in for repogen's output: an emit interface
			// containing the to-be-mocked methods, anchored to a
			// source struct via Origin.
			srcPkg := &node.Package{Name: "blog", Path: "example.com/blog"}
			article := &node.Struct{
				Name:    "Article",
				Package: srcPkg.Path,
				BaseNode: node.BaseNode{
					SourcePos: position.Pos{File: "blog/article.go", Line: 1},
				},
			}
			srcPkg.Structs = append(srcPkg.Structs, article)
			if err := s.Nodes().AddPackage(srcPkg); err != nil {
				t.Fatalf("Nodes.AddPackage: %v", err)
			}

			epkg := &emit.Package{Name: "blog", Path: "example.com/blog"}
			epkg.Interfaces = []*emit.Interface{{
				BaseEmit: emit.BaseEmit{OriginNode: article},
				Name:     "ArticleRepository",
				Package:  epkg.Path,
			}}
			if err := s.Emit().AddPackage(epkg); err != nil {
				t.Fatalf("Emit.AddPackage: %v", err)
			}

			p := configuredPlugin(t)
			ctx := &plugin.GeneratorContext{
				Store: s, Reader: store.NewReader(s), Diag: diag.New(),
			}
			if err := p.Generate(ctx); err != nil {
				t.Fatalf("Generate: %v", err)
			}

			wantPath := epkg.Path + mockgen.TestPackageSuffix
			emitPkg, ok := s.Emit().Packages().ByQName(wantPath)
			if !ok {
				t.Fatalf(
					"expected emit.Package keyed by %q; got %+v",
					wantPath,
					s.Emit().Packages().Items(),
				)
			}
			var mock *emit.Struct
			for _, st := range emitPkg.Structs {
				if st.Name == "ArticleRepositoryMock" {
					mock = st
					break
				}
			}
			if mock == nil {
				t.Fatalf("emit.Package missing ArticleRepositoryMock; got %+v", emitPkg.Structs)
			}
			if mock.Target != (emit.Target{}) {
				t.Fatalf("Target should be zero until Layout composes it; got %+v", mock.Target)
			}
			if mock.Origin() != article {
				t.Fatalf(
					"Origin should propagate from the upstream interface so Layout can resolve the Target",
				)
			}
		},
	)
}

// TestGenerate_SuppressEmitInterface covers the `-gen:mock` opt-out
// branch on emit-store interfaces: an emit interface carrying the
// negated directive is skipped even though mockgen otherwise mocks
// every emit interface.
func TestGenerate_SuppressEmitInterface(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		directives []*directive.Directive
		want       bool
	}{
		{
			name:       "no directive — mock is emitted",
			directives: nil,
			want:       true,
		},
		{
			name:       "negated directive — mock is suppressed",
			directives: []*directive.Directive{{Name: mockgen.DirectiveName, Negated: true}},
			want:       false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := configuredPlugin(t)
			s := store.New()
			pkg := &emit.Package{Name: "gen", Path: "gen"}
			pkg.Interfaces = []*emit.Interface{{
				BaseEmit: emit.BaseEmit{DirectiveList: tc.directives},
				Name:     "Probe", Package: "gen",
			}}
			if err := s.Emit().AddPackage(pkg); err != nil {
				t.Fatalf("EmitView.AddPackage: %v", err)
			}
			ctx := &plugin.GeneratorContext{
				Store: s, Reader: store.NewReader(s), Diag: diag.New(),
			}
			if err := p.Generate(ctx); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			got := emitContainsStruct(s, "ProbeMock")
			if got != tc.want {
				t.Fatalf("emit-store contains ProbeMock = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerate_OptInSourceInterface covers the `+gen:mock`
// opt-in branch on source-side interfaces: source interfaces only
// get mocked when they explicitly carry the directive.
func TestGenerate_OptInSourceInterface(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		directives []*directive.Directive
		want       bool
	}{
		{
			name:       "positive directive — mock is emitted",
			directives: []*directive.Directive{{Name: mockgen.DirectiveName}},
			want:       true,
		},
		{
			name:       "no directive — mock is suppressed",
			directives: nil,
			want:       false,
		},
		{
			name:       "negated directive — mock is suppressed",
			directives: []*directive.Directive{{Name: mockgen.DirectiveName, Negated: true}},
			want:       false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := configuredPlugin(t)
			s := store.New()
			pkg := &node.Package{Name: "synth", Path: "example.com/synth"}
			pkg.Interfaces = []*node.Interface{{
				BaseNode: node.BaseNode{DirectiveList: tc.directives},
				Name:     "Probe", Package: "example.com/synth",
				Methods: []*node.Method{{Name: "Ping"}},
			}}
			if err := s.Nodes().AddPackage(pkg); err != nil {
				t.Fatalf("NodeView.AddPackage: %v", err)
			}
			ctx := &plugin.GeneratorContext{
				Store: s, Reader: store.NewReader(s), Diag: diag.New(),
			}
			if err := p.Generate(ctx); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			got := emitContainsStruct(s, "ProbeMock")
			if got != tc.want {
				t.Fatalf("emit-store contains ProbeMock = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerate_DispatchBodyZeroReturn covers the zero-return
// branch in the dispatch-body builder: a method with no returns
// renders as a bare `m.<Name>Func(...)` expression statement rather
// than `return m.<Name>Func(...)`.
func TestGenerate_DispatchBodyZeroReturn(t *testing.T) {
	t.Parallel()

	p := configuredPlugin(t)
	s := store.New()
	pkg := &node.Package{Name: "synth", Path: "example.com/synth"}
	pkg.Interfaces = []*node.Interface{{
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{{Name: mockgen.DirectiveName}},
		},
		Name: "Notifier", Package: "example.com/synth",
		Methods: []*node.Method{{
			Name: "Notify",
			Params: []*node.Param{
				{Name: "msg", Type: &node.TypeRef{TypeKind: node.TypeRefNamed, Name: "string"}},
			},
		}},
	}}
	if err := s.Nodes().AddPackage(pkg); err != nil {
		t.Fatalf("NodeView.AddPackage: %v", err)
	}
	ctx := &plugin.GeneratorContext{
		Store: s, Reader: store.NewReader(s), Diag: diag.New(),
	}
	if err := p.Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// The emitted mock lives under the source package's import
	// path plus the test-package suffix — mockgen drops every mock
	// in an external `<srcPkg>_test` package by default.
	mock, ok := s.Emit().Structs().ByQName(pkg.Path + mockgen.TestPackageSuffix + ".NotifierMock")
	if !ok {
		t.Fatalf("emit store missing NotifierMock; got %+v", s.Emit().Structs().Items())
	}
	if len(mock.Methods) != 1 {
		t.Fatalf("expected one mock method; got %d", len(mock.Methods))
	}
	body := mock.Methods[0].Body
	if len(body) != 1 {
		t.Fatalf("expected one body statement; got %d", len(body))
	}
	if body[0].StmtKind != emit.StmtExpr {
		t.Fatalf("zero-return body must be a bare expression stmt; got %v", body[0].StmtKind)
	}
}

// TestGenerate_AnonymousParamsGetNames covers the anonymous-param
// rewriting branch: a source method with un-named parameters still
// produces a syntactically-valid mock body by naming the call sites
// `arg0`, `arg1`, …
func TestGenerate_AnonymousParamsGetNames(t *testing.T) {
	t.Parallel()

	p := configuredPlugin(t)
	s := store.New()
	pkg := &node.Package{Name: "synth", Path: "example.com/synth"}
	pkg.Interfaces = []*node.Interface{{
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{{Name: mockgen.DirectiveName}},
		},
		Name: "AnonProbe", Package: "example.com/synth",
		Methods: []*node.Method{{
			Name: "Do",
			Params: []*node.Param{
				{Type: &node.TypeRef{TypeKind: node.TypeRefNamed, Name: "int"}},
				{Type: &node.TypeRef{TypeKind: node.TypeRefNamed, Name: "string"}},
			},
			Returns: []*node.TypeRef{{TypeKind: node.TypeRefNamed, Name: "error"}},
		}},
	}}
	if err := s.Nodes().AddPackage(pkg); err != nil {
		t.Fatalf("NodeView.AddPackage: %v", err)
	}
	ctx := &plugin.GeneratorContext{
		Store: s, Reader: store.NewReader(s), Diag: diag.New(),
	}
	if err := p.Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	mock, ok := s.Emit().Structs().ByQName(pkg.Path + mockgen.TestPackageSuffix + ".AnonProbeMock")
	if !ok {
		t.Fatalf("emit store missing AnonProbeMock")
	}
	method := mock.Methods[0]
	if got, want := method.Params[0].Name, "arg0"; got != want {
		t.Fatalf("first anonymous param name = %q, want %q", got, want)
	}
	if got, want := method.Params[1].Name, "arg1"; got != want {
		t.Fatalf("second anonymous param name = %q, want %q", got, want)
	}
}

// configuredPlugin returns a fresh mockgen plugin with framework
// defaults applied so synthetic-store tests can call Generate
// directly without going through the pipeline's option-decode
// plumbing. Routing is owned by the framework's routing layer and
// is not part of the plugin's option surface.
func configuredPlugin(t *testing.T) *mockgen.Plugin {
	t.Helper()
	p := mockgen.New()
	if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
		t.Fatalf("SetOptions: %v", err)
	}
	return p
}

// emitContainsStruct reports whether the emit store carries a
// struct whose Name equals want.
func emitContainsStruct(s *store.Store, want string) bool {
	for _, st := range s.Emit().Structs().Items() {
		if st.Name == want {
			return true
		}
	}
	return false
}

// sinkBody returns the rendered body for filename under the
// configured output package. Fails the test when the entry is
// missing.
func sinkBody(t *testing.T, s sink.Sink, filename string) string {
	t.Helper()
	mem, ok := s.(*sink.Memory)
	if !ok {
		t.Fatalf("sink is not *sink.Memory; got %T", s)
	}
	for target, body := range mem.Files() {
		if target.Filename == filename && target.Package == outputPackage {
			return string(body)
		}
	}
	t.Fatalf("sink missing %q under package %q", filename, outputPackage)
	return ""
}

// planFor builds the demopipe pipeline plan via DryRun and returns
// the resolved plugin order for the generator phase so the
// repogen-before-mockgen ordering assertion can inspect it.
func planFor(t *testing.T) []string {
	t.Helper()
	result := demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{mockgen.New(), repogen.New()},
		Backend:       backend_golang.New(),
		Layout:        pipeline.LayoutCentralised,
		OutputPackage: outputPackage,
	})
	if result.RunErr != nil {
		t.Fatalf("planFor: pipeline Run: %v", result.RunErr)
	}
	// Walk the emit store's structs in insertion order to derive
	// the de-facto plugin order — Repo structs were emitted before
	// any *Mock struct iff repogen ran first.
	var order []string
	seen := map[string]bool{}
	for _, st := range result.Store.Emit().Structs().Items() {
		switch {
		case strings.HasSuffix(st.Name, "Mock"):
			if !seen[mockgen.Name] {
				order = append(order, mockgen.Name)
				seen[mockgen.Name] = true
			}
		case strings.HasSuffix(st.Name, "Repo"):
			if !seen[repogen.Name] {
				order = append(order, repogen.Name)
				seen[repogen.Name] = true
			}
		}
	}
	return order
}

// indexOf returns the index of want in s, or -1 when absent. Used
// by the plan-order assertion to compare the relative position of
// repogen and mockgen.
func indexOf(s []string, want string) int {
	for i, v := range s {
		if v == want {
			return i
		}
	}
	return -1
}

// TestConformance runs the framework's plugin-conformance suite
// against this package's plugin. The suite pins the standard
// framework contracts (stable Name, role-interface compliance,
// deterministic capability ordering, unique directive schema
// names, non-empty Versioned version) so a regression on any
// of them surfaces here before downstream tests trip over it.
func TestConformance(t *testing.T) {
	t.Parallel()
	plugintest.RunSuite(t, mockgen.New())
}
