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
	"go.thesmos.sh/eidos/node"
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
		article := sinkBody(t, result.Sink, "article.go")
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
	article := sinkBody(t, result.Sink, "article.go")
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

// TestGenerate_AlongsideSourceLayout covers the default layout: an
// unconfigured plugin (OutputPackage empty) emits one decl per
// source struct with [emit.Target.Dir] left empty and a
// `<src>_repo.go` filename so the pipeline router fills the Dir
// from the source's directory at the routing phase.
func TestGenerate_AlongsideSourceLayout(t *testing.T) {
	t.Parallel()

	t.Run("empty OutputPackage drops decls alongside the source", func(t *testing.T) {
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
		emitPkg, _ := pkgs.ByQName(repogen.Name + ":" + srcPkg.Path)
		if emitPkg == nil {
			t.Fatalf("expected emit.Package keyed by plugin-namespaced source path; got %v", pkgs)
		}
		if emitPkg.Name != srcPkg.Name {
			t.Fatalf("emit.Package.Name = %q, want %q (matches source pkg name)", emitPkg.Name, srcPkg.Name)
		}
		if len(emitPkg.Interfaces) != 1 {
			t.Fatalf("expected one emitted interface; got %d", len(emitPkg.Interfaces))
		}
		iface := emitPkg.Interfaces[0]
		if iface.Target.Dir != "" {
			t.Fatalf("Target.Dir should be empty so the router fills it; got %q", iface.Target.Dir)
		}
		if iface.Target.Filename != "probe"+repogen.FilenameSuffix {
			t.Fatalf("Target.Filename = %q, want probe%s", iface.Target.Filename, repogen.FilenameSuffix)
		}
		if iface.Target.Package != "users" {
			t.Fatalf("Target.Package = %q, want users", iface.Target.Package)
		}
		if iface.Origin() != src {
			t.Fatalf("Origin should be the source struct so the router can resolve Dir")
		}
	})
}

// runFixture builds the demopipe harness with repogen engaged and
// the supplied option overrides applied. Returns the captured
// pipeline result with sink + diag + store ready for assertions.
func runFixture(t *testing.T, extraOpts map[string]string) demopipe.Result {
	t.Helper()
	opts := map[string]string{"output_package": outputPackage}
	maps.Copy(opts, extraOpts)
	return demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{repogen.New()},
		Backend:       backend_golang.New(),
		PluginOptions: map[string]map[string]string{repogen.Name: opts},
	})
}

// configuredPlugin returns a fresh repogen plugin with OutputPackage
// applied so synthetic-store tests can call Generate directly
// without going through the pipeline's option-decode plumbing.
func configuredPlugin(t *testing.T) *repogen.Plugin {
	t.Helper()
	p := repogen.New()
	o := opt.New(p.OptionsSchema(), map[string]string{"output_package": outputPackage})
	if err := p.SetOptions(o); err != nil {
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
