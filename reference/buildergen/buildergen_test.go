// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package buildergen_test

import (
	"maps"
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/eidostest/demopipe"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/buildergen"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// outputPackage is the canonical destination package for emitted
// builder decls in the tests. Aligned with repogen's convention so
// the composition acceptance criterion (one file per source struct
// when both generators share OutputPackage) can be exercised
// later.
const outputPackage = "gen"

// TestPluginShape pins the plugin's public-contract surface so
// rename / drop accidents surface at PR review time.
func TestPluginShape(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented identifier", func(t *testing.T) {
		t.Parallel()
		if got := buildergen.New().Name(); got != buildergen.Name {
			t.Fatalf("Name = %q, want %q", got, buildergen.Name)
		}
	})

	t.Run("implements Generator, CapabilityProvider, OptionsProvider, DirectiveProvider", func(t *testing.T) {
		t.Parallel()
		p := buildergen.New()
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

	t.Run("Provides advertises the builder capability", func(t *testing.T) {
		t.Parallel()
		got := buildergen.New().Provides()
		if len(got) != 1 || got[0] != buildergen.Capability {
			t.Fatalf("Provides = %+v, want [%q]", got, buildergen.Capability)
		}
	})

	t.Run("Directives returns the builder schema", func(t *testing.T) {
		t.Parallel()
		schemas := buildergen.New().Directives()
		if len(schemas) != 1 {
			t.Fatalf("expected one schema; got %d", len(schemas))
		}
		if schemas[0].Name != buildergen.DirectiveName {
			t.Fatalf("schema name = %q, want %q", schemas[0].Name, buildergen.DirectiveName)
		}
		if !schemas[0].AllowNegated {
			t.Fatalf("schema must allow the negated form for opt-out support")
		}
	})
}

// TestGenerate_EndToEnd runs the plugin against the demoproject
// fixture and asserts the builder surface — type, setters, Build —
// reaches the rendered sink for every `+gen:builder` target.
func TestGenerate_EndToEnd(t *testing.T) {
	t.Parallel()

	result := runFixture(t, nil)
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}

	t.Run("annotated structs each produce a Builder type", func(t *testing.T) {
		t.Parallel()
		for _, want := range []string{"ArticleBuilder", "UserBuilder", "CommentBuilder"} {
			if !emitContainsStruct(result.Store, want) {
				t.Fatalf("expected emit store to contain %q", want)
			}
		}
	})

	t.Run("rendered Article builder carries setters for every exported field plus Build", func(t *testing.T) {
		t.Parallel()
		body := sinkBody(t, result.Sink, "article.go")
		for _, want := range []string{
			"type ArticleBuilder struct",
			"func (b *ArticleBuilder) WithID(value [16]byte) *ArticleBuilder",
			"func (b *ArticleBuilder) WithTitle(value string) *ArticleBuilder",
			"func (b *ArticleBuilder) WithStatus(value blog.Status) *ArticleBuilder",
			"func (b *ArticleBuilder) WithTags(value []string) *ArticleBuilder",
			"func (b *ArticleBuilder) WithMeta(value map[string]string) *ArticleBuilder",
			"func (b *ArticleBuilder) WithAuthor(value *blog.User) *ArticleBuilder",
			"func (b *ArticleBuilder) Build() *blog.Article",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("ArticleBuilder rendered output missing %q; got:\n%s", want, body)
			}
		}
	})

	t.Run("Comment builder exposes its exported fields including the generic Box embed", func(t *testing.T) {
		t.Parallel()
		body := sinkBody(t, result.Sink, "comment.go")
		for _, want := range []string{
			"type CommentBuilder struct",
			"func (b *CommentBuilder) Build() *blog.Comment",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("CommentBuilder rendered output missing %q; got:\n%s", want, body)
			}
		}
	})
}

// TestGenerate_DirectiveGating pins the directive-driven emission
// gate: positive directive emits, negated directive suppresses,
// missing directive defaults to no-emission. Each case drives
// buildergen against a synthetic node store so the gate is
// exercised independently of the demoproject fixture.
func TestGenerate_DirectiveGating(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		directive *directive.Directive
		want      bool
	}{
		{
			name:      "positive directive emits",
			directive: &directive.Directive{Name: buildergen.DirectiveName},
			want:      true,
		},
		{
			name:      "negated directive suppresses",
			directive: &directive.Directive{Name: buildergen.DirectiveName, Negated: true},
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
			got := emitContainsStruct(s, "ProbeBuilder")
			if got != tc.want {
				t.Fatalf("emit-store contains builder = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGenerate_FieldTypeCoverage exercises [buildergen.refFromNode]
// through the public Generate path: a synthetic struct with one
// field per supported TypeRef variant produces a builder whose
// `With<Field>` setter takes the matching emit ref. Anchors the
// type-conversion helper against future TypeRefKind additions.
func TestGenerate_FieldTypeCoverage(t *testing.T) {
	t.Parallel()

	p := configuredPlugin(t)
	s := store.New()
	pkg := &node.Package{
		Name: "synth", Path: "example.com/synth",
		Structs: []*node.Struct{{
			Name:    "Probe",
			Package: "example.com/synth",
			BaseNode: node.BaseNode{
				DirectiveList: []*directive.Directive{
					{Name: buildergen.DirectiveName},
				},
			},
			Fields: []*node.Field{
				{Name: "Builtin", Type: namedRef("", "string")},
				{Name: "Pointer", Type: pointerRef(namedRef("example.com/synth", "Probe"))},
				{Name: "Slice", Type: sliceRef(namedRef("", "int"))},
				{Name: "Map", Type: mapRef(namedRef("", "string"), namedRef("", "int"))},
				{Name: "External", Type: namedRef("example.com/extras", "UUID")},
				{Name: "Generic", Type: &node.TypeRef{TypeKind: node.TypeRefTypeParam, Name: "T"}},
				{Name: "URLHandler", Type: namedRef("", "string")},
				{Name: "unexported", Type: namedRef("", "string")},
			},
		}},
	}
	if err := s.Nodes().AddPackage(pkg); err != nil {
		t.Fatalf("NodeView.AddPackage: %v", err)
	}
	ctx := &plugin.GeneratorContext{
		Store: s, Reader: store.NewReader(s), Diag: diag.New(),
	}
	if err := p.Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	probe, ok := s.Emit().Structs().ByQName(outputPackage + ".ProbeBuilder")
	if !ok {
		t.Fatalf("emit store missing ProbeBuilder; got %+v", s.Emit().Structs().Items())
	}
	if len(probe.Fields) != 7 {
		t.Fatalf("expected 7 exported fields on the builder; got %d (%+v)", len(probe.Fields), probe.Fields)
	}
	wantNames := map[string]bool{
		"builtin": false, "pointer": false, "slice": false,
		"map": false, "external": false, "generic": false,
		"urlHandler": false,
	}
	for _, f := range probe.Fields {
		wantNames[f.Name] = true
	}
	for k, seen := range wantNames {
		if !seen {
			t.Fatalf("builder is missing the %q accumulator field", k)
		}
	}
	// One `With<Field>` per exported source field + Build.
	if want := 7 + 1; len(probe.Methods) != want {
		t.Fatalf("builder method count = %d, want %d (one per field plus Build)", len(probe.Methods), want)
	}
}

// TestGenerate_MissingOutputPackage covers the option-precondition
// branch: an unconfigured plugin returns the sentinel error rather
// than emitting decls with an empty Target.
func TestGenerate_MissingOutputPackage(t *testing.T) {
	t.Parallel()

	p := buildergen.New()
	s := store.New()
	ctx := &plugin.GeneratorContext{
		Store: s, Reader: store.NewReader(s), Diag: diag.New(),
	}
	err := p.Generate(ctx)
	if err == nil {
		t.Fatalf("Generate without OutputPackage should error; got nil")
	}
	if !strings.Contains(err.Error(), "OutputPackage") {
		t.Fatalf("error should reference OutputPackage; got %q", err)
	}
}

// runFixture builds the demopipe harness with buildergen engaged
// and the supplied option overrides applied.
func runFixture(t *testing.T, extraOpts map[string]string) demopipe.Result {
	t.Helper()
	opts := map[string]string{"output_package": outputPackage}
	maps.Copy(opts, extraOpts)
	return demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{buildergen.New()},
		Backend:       backend_golang.New(),
		PluginOptions: map[string]map[string]string{buildergen.Name: opts},
	})
}

// configuredPlugin returns a fresh buildergen plugin with the
// minimum required options applied so synthetic-store tests can
// call Generate directly without going through the pipeline's
// option-decode plumbing.
func configuredPlugin(t *testing.T) *buildergen.Plugin {
	t.Helper()
	p := buildergen.New()
	o := opt.New(p.OptionsSchema(), map[string]string{"output_package": outputPackage})
	if err := p.SetOptions(o); err != nil {
		t.Fatalf("SetOptions: %v", err)
	}
	return p
}

// syntheticPackage constructs an in-memory [node.Package] wrapping a
// single struct named name with the supplied directive attached
// (nil leaves the struct undirected).
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

// namedRef constructs a Named [node.TypeRef] for the supplied
// package + name. Empty pkg yields a builtin / unqualified ref.
func namedRef(pkg, name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Package: pkg, Name: name}
}

// pointerRef wraps elem in a Pointer [node.TypeRef].
func pointerRef(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: elem}
}

// sliceRef wraps elem in a Slice [node.TypeRef].
func sliceRef(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: elem}
}

// mapRef constructs a Map [node.TypeRef] with the supplied key
// and value types.
func mapRef(key, value *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefMap, MapKey: key, MapValue: value}
}
