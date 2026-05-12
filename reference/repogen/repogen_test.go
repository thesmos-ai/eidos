// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package repogen_test

import (
	"maps"
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/demopipe"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// outputPackage is the package name + dir the tests route emitted
// repository decls into. Centralised so every test asserts against
// the same Target.
const outputPackage = "gen"

// TestPluginShape covers the plugin's public-contract surface:
// stable identifier, declared role interfaces, and directive
// schema. Pins the contract at PR time so accidental rename / drop
// surfaces immediately.
func TestPluginShape(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented identifier", func(t *testing.T) {
		t.Parallel()
		if got := repogen.New().Name(); got != repogen.Name {
			t.Fatalf("Name = %q, want %q", got, repogen.Name)
		}
	})

	t.Run("implements Generator, CapabilityProvider, OptionsProvider, DirectiveProvider", func(t *testing.T) {
		t.Parallel()
		p := repogen.New()
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
	})

	t.Run("Provides advertises the repository capability", func(t *testing.T) {
		t.Parallel()
		got := repogen.New().Provides()
		if len(got) != 1 || got[0] != repogen.Capability {
			t.Fatalf("Provides = %+v, want [%q]", got, repogen.Capability)
		}
	})

	t.Run("Directives returns the repo schema", func(t *testing.T) {
		t.Parallel()
		schemas := repogen.New().Directives()
		if len(schemas) != 1 {
			t.Fatalf("expected one schema; got %d", len(schemas))
		}
		if schemas[0].Name != repogen.DirectiveName {
			t.Fatalf("schema name = %q, want %q", schemas[0].Name, repogen.DirectiveName)
		}
		if !schemas[0].AllowNegated {
			t.Fatalf("schema must allow the negated form for opt-out support")
		}
	})
}

// TestGenerate_EndToEnd exercises the full pipeline path: parse
// demoproject, run repogen against the source structs, render the
// emitted decls through the Go backend, and verify the canonical
// repository surface lands in the in-memory sink.
func TestGenerate_EndToEnd(t *testing.T) {
	t.Parallel()

	result := runFixture(t, nil)
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}

	t.Run("annotated structs produce a Repository interface plus implementing struct", func(t *testing.T) {
		t.Parallel()
		for _, want := range []string{"ArticleRepository", "ArticleRepo", "UserRepository", "UserRepo"} {
			if !emitContainsType(result.Store, want) {
				t.Fatalf("expected emit store to contain %q", want)
			}
		}
	})

	t.Run("Comment is unannotated and therefore not emitted", func(t *testing.T) {
		t.Parallel()
		for _, unwanted := range []string{"CommentRepository", "CommentRepo"} {
			if emitContainsType(result.Store, unwanted) {
				t.Fatalf("did not expect emit store to contain %q (Comment lacks +gen:repo)", unwanted)
			}
		}
	})

	t.Run("rendered sink output contains the canonical CRUD method set", func(t *testing.T) {
		t.Parallel()
		article := sinkBody(t, result.Sink, "article"+repogen.FilenameSuffix)
		for _, want := range []string{
			"type ArticleRepository interface",
			"Get(ctx context.Context, id string) (*blog.Article, error)",
			"List(ctx context.Context) ([]*blog.Article, error)",
			"Save(ctx context.Context, value *blog.Article) error",
			"Delete(ctx context.Context, id string) error",
			"type ArticleRepo struct",
		} {
			if !strings.Contains(article, want) {
				t.Fatalf("article.go missing %q; got:\n%s", want, article)
			}
		}
	})
}

// TestGenerate_NamingCamel covers the Naming=Camel option: every
// emitted identifier — both the type names and the method names —
// is lower-cased on the first rune so callers can keep the
// repository internal to the consuming package.
func TestGenerate_NamingCamel(t *testing.T) {
	t.Parallel()

	result := runFixture(t, map[string]string{"naming": repogen.NamingCamel})
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}
	article := sinkBody(t, result.Sink, "article"+repogen.FilenameSuffix)
	for _, want := range []string{
		"type articleRepository interface",
		"type articleRepo struct",
		"get(ctx context.Context, id string)",
		"save(ctx context.Context, value *blog.Article)",
	} {
		if !strings.Contains(article, want) {
			t.Fatalf("Camel naming missed %q in:\n%s", want, article)
		}
	}
	for _, unwanted := range []string{"ArticleRepository", "ArticleRepo"} {
		if strings.Contains(article, unwanted) {
			t.Fatalf("Camel naming leaked Pascal identifier %q in:\n%s", unwanted, article)
		}
	}
}

// TestGenerate_DirectiveGating pins the directive-driven emission
// gate: positive directive emits, negated directive suppresses,
// and a missing directive defaults to no-emission. Each case drives
// repogen against a synthetic node store so the gate is exercised
// independently of the demoproject fixture.
func TestGenerate_DirectiveGating(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		directive *directive.Directive
		want      bool
	}{
		{
			name:      "positive directive emits",
			directive: &directive.Directive{Name: repogen.DirectiveName},
			want:      true,
		},
		{
			name:      "negated directive suppresses",
			directive: &directive.Directive{Name: repogen.DirectiveName, Negated: true},
			want:      false,
		},
		{
			name:      "missing directive suppresses",
			directive: nil,
			want:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := configuredPlugin(t)
			s := store.New()
			if err := s.Nodes().AddPackage(syntheticPackage("Probe", tc.directive)); err != nil {
				t.Fatalf("NodeView.AddPackage: %v", err)
			}
			ctx := &plugin.GeneratorContext{
				Store: s, Reader: store.NewReader(s), Diag: diag.New(),
			}
			if err := p.Generate(ctx); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			got := emitContainsType(s, "ProbeRepository") || emitContainsType(s, "ProbeRepo")
			if got != tc.want {
				t.Fatalf("emit-store contains repo decls = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerate_HonoursScope pins the routing-layer scope contract
// from the consumer side: when the pipeline carries a
// target-symbol predicate, repogen emits decls only for source
// structs matching the predicate. The contract requires plugins
// to iterate via scoped reader queries — iterating raw struct
// slices off [node.Package] would bypass the scope filter and
// produce out-of-scope output.
func TestGenerate_HonoursScope(t *testing.T) {
	t.Parallel()

	result := demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{repogen.New()},
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

	t.Run("Article in scope produces its repository surface", func(t *testing.T) {
		t.Parallel()
		for _, want := range []string{"ArticleRepository", "ArticleRepo"} {
			if !emitContainsType(result.Store, want) {
				t.Fatalf("expected emit store to contain %q under -target Article", want)
			}
		}
	})

	t.Run("User out of scope produces no repository surface", func(t *testing.T) {
		t.Parallel()
		for _, unwanted := range []string{"UserRepository", "UserRepo"} {
			if emitContainsType(result.Store, unwanted) {
				t.Fatalf("did not expect %q under -target Article scope", unwanted)
			}
		}
	})
}

// TestGenerate_LeavesTargetForLayout pins the routing-layer
// contract: the plugin emits decls with Origin set but Target
// fields untouched. The framework's Layout phase composes
// Target.Dir / Filename / Package / ImportPath downstream — the
// plugin never constructs an [emit.Target] literal.
func TestGenerate_LeavesTargetForLayout(t *testing.T) {
	t.Parallel()

	t.Run("emitted decls carry Origin but no plugin-stamped Target", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		srcPkg := &node.Package{Name: "users", Path: "example.com/users"}
		src := &node.Struct{
			Name:    "Probe",
			Package: srcPkg.Path,
			BaseNode: node.BaseNode{
				SourcePos:     position.Pos{File: "users/probe.go", Line: 1},
				DirectiveList: []*directive.Directive{{Name: repogen.DirectiveName}},
			},
		}
		srcPkg.Structs = append(srcPkg.Structs, src)
		if err := s.Nodes().AddPackage(srcPkg); err != nil {
			t.Fatalf("AddPackage: %v", err)
		}

		p := repogen.New()
		if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
			t.Fatalf("SetOptions: %v", err)
		}
		ctx := &plugin.GeneratorContext{
			Store: s, Reader: store.NewReader(s), Diag: diag.New(),
		}
		if err := p.Generate(ctx); err != nil {
			t.Fatalf("Generate: %v", err)
		}

		pkgs := s.Emit().Packages()
		if pkgs.Len() != 1 {
			t.Fatalf("expected one emit.Package; got %d", pkgs.Len())
		}
		emitPkg, _ := pkgs.ByQName(srcPkg.Path)
		if emitPkg == nil {
			t.Fatalf("expected emit.Package keyed by source path %q; got %v", srcPkg.Path, pkgs)
		}
		if len(emitPkg.Interfaces) != 1 {
			t.Fatalf("expected one emitted interface; got %d", len(emitPkg.Interfaces))
		}
		iface := emitPkg.Interfaces[0]
		if iface.Target != (emit.Target{}) {
			t.Fatalf("Target should be zero until Layout composes it; got %+v", iface.Target)
		}
		if iface.Origin() != src {
			t.Fatalf("Origin should be the source struct so Layout can resolve the Target")
		}
	})
}

// runFixture builds the demopipe harness with repogen engaged and
// the supplied option overrides applied. Returns the captured
// pipeline result with sink + diag + store ready for assertions.
//
// The harness pins centralised layout + a shared output package via
// the routing-layer surface ([pipeline.Builder.WithOutputLayout] /
// [pipeline.Builder.WithOutputPackage]) so every rendered file
// lands in the configured package directory regardless of source
// location.
func runFixture(t *testing.T, extraOpts map[string]string) demopipe.Result {
	t.Helper()
	opts := map[string]string{}
	maps.Copy(opts, extraOpts)
	runOpts := demopipe.RunOptions{
		Generators:    []plugin.Generator{repogen.New()},
		Backend:       backend_golang.New(),
		Layout:        pipeline.LayoutCentralised,
		OutputPackage: outputPackage,
	}
	if len(opts) > 0 {
		runOpts.PluginOptions = map[string]map[string]string{repogen.Name: opts}
	}
	return demopipe.Run(t, runOpts)
}

// configuredPlugin returns a fresh repogen plugin with the
// framework's defaults applied so synthetic-store tests can call
// Generate directly without going through the pipeline's
// option-decode plumbing. Routing is owned by the framework's
// routing layer and is not part of the plugin's option surface.
func configuredPlugin(t *testing.T) *repogen.Plugin {
	t.Helper()
	p := repogen.New()
	if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
		t.Fatalf("SetOptions: %v", err)
	}
	return p
}

// syntheticPackage constructs an in-memory [node.Package] wrapping a
// single struct named name with the supplied directive attached
// (nil leaves the struct undirected). Used by the directive-gating
// test to exercise repogen against a minimal store without touching
// the demoproject fixture.
func syntheticPackage(name string, d *directive.Directive) *node.Package {
	s := &node.Struct{Name: name, Package: "example.com/synth"}
	if d != nil {
		s.DirectiveList = append(s.DirectiveList, d)
	}
	return &node.Package{
		Name:    "synth",
		Path:    "example.com/synth",
		Structs: []*node.Struct{s},
	}
}

// emitContainsType reports whether the emit store carries an
// interface or struct whose Name equals want — the indicator the
// generator emitted (or skipped) the expected decl.
func emitContainsType(s *store.Store, want string) bool {
	for _, i := range s.Emit().Interfaces().Items() {
		if i.Name == want {
			return true
		}
	}
	for _, st := range s.Emit().Structs().Items() {
		if st.Name == want {
			return true
		}
	}
	return false
}

// sinkBody returns the rendered body for filename under the
// configured output package. Fails the test if the entry is
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
