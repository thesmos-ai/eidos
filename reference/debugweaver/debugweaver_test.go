// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package debugweaver_test

import (
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/eidostest/demopipe"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/debugweaver"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// outputPackage is the destination package for the foundation
// generator the weavers run against in the end-to-end test.
const outputPackage = "gen"

// TestPluginShape pins the plugin's public-contract surface.
func TestPluginShape(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented identifier", func(t *testing.T) {
		t.Parallel()
		if got := debugweaver.New().Name(); got != debugweaver.Name {
			t.Fatalf("Name = %q, want %q", got, debugweaver.Name)
		}
	})

	t.Run("implements Generator, CapabilityProvider, DirectiveProvider", func(t *testing.T) {
		t.Parallel()
		p := debugweaver.New()
		if _, ok := any(p).(plugin.Generator); !ok {
			t.Fatalf("plugin must implement plugin.Generator")
		}
		if _, ok := any(p).(plugin.CapabilityProvider); !ok {
			t.Fatalf("plugin must implement plugin.CapabilityProvider")
		}
		if _, ok := any(p).(plugin.DirectiveProvider); !ok {
			t.Fatalf("plugin must implement plugin.DirectiveProvider")
		}
	})

	t.Run("Provides advertises the trace capability", func(t *testing.T) {
		t.Parallel()
		got := debugweaver.New().Provides()
		if len(got) != 1 || got[0] != debugweaver.Capability {
			t.Fatalf("Provides = %+v, want [%q]", got, debugweaver.Capability)
		}
	})
}

// TestGenerate_WeavesEveryEmitMethod runs repogen + debug-weaver
// against the demoproject fixture and asserts every repogen-emitted
// method picks up the entry-trace contribution in its rendered
// body.
func TestGenerate_WeavesEveryEmitMethod(t *testing.T) {
	t.Parallel()

	result := demopipe.Run(t, demopipe.RunOptions{
		Generators: []plugin.Generator{repogen.New(), debugweaver.New()},
		Backend:    backend_golang.New(),
		PluginOptions: map[string]map[string]string{
			repogen.Name: {"output_package": outputPackage},
		},
	})
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}

	body := sinkBody(t, result.Sink, "article.go")
	for _, want := range []string{
		`log.Printf("debug: ArticleRepo.Get entered")`,
		`log.Printf("debug: ArticleRepo.List entered")`,
		`log.Printf("debug: ArticleRepo.Save entered")`,
		`log.Printf("debug: ArticleRepo.Delete entered")`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("article.go missing entry trace %q; got:\n%s", want, body)
		}
	}
}

// TestGenerate_SuppressionByDirective covers the per-method
// opt-out path: an emit method carrying `-gen:debug` does not get
// the trace contribution, while a sibling method without the
// directive does.
func TestGenerate_SuppressionByDirective(t *testing.T) {
	t.Parallel()

	s := store.New()
	pkg := &emit.Package{Name: outputPackage, Path: outputPackage}
	host := &emit.Struct{Name: "Service", Package: outputPackage}
	suppressed := &emit.Method{
		Name:  "Skipped",
		Owner: host,
		BaseEmit: emit.BaseEmit{
			DirectiveList: []*directive.Directive{
				{Name: debugweaver.DirectiveName, Negated: true},
			},
		},
	}
	included := &emit.Method{Name: "Traced", Owner: host}
	host.Methods = []*emit.Method{suppressed, included}
	pkg.Structs = []*emit.Struct{host}
	if err := s.Emit().AddPackage(pkg); err != nil {
		t.Fatalf("EmitView.AddPackage: %v", err)
	}
	ctx := &plugin.GeneratorContext{
		Store: s, Reader: store.NewReader(s), Diag: diag.New(),
	}
	if err := debugweaver.New().Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if suppressed.Prebody().Len() != 0 {
		t.Fatalf("suppressed method must not receive a trace contribution; got %d entries", suppressed.Prebody().Len())
	}
	if included.Prebody().Len() != 1 {
		t.Fatalf("included method must receive one trace contribution; got %d entries", included.Prebody().Len())
	}
}

// sinkBody returns the rendered body for filename under the
// configured output package.
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
