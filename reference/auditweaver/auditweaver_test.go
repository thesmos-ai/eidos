// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package auditweaver_test

import (
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/eidostest/demopipe"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/auditweaver"
	"go.thesmos.sh/eidos/reference/debugweaver"
	"go.thesmos.sh/eidos/reference/repogen"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// outputPackage is the destination package for the foundation
// generator the weavers run against.
const outputPackage = "gen"

// TestPluginShape pins the plugin's public-contract surface.
func TestPluginShape(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented identifier", func(t *testing.T) {
		t.Parallel()
		if got := auditweaver.New().Name(); got != auditweaver.Name {
			t.Fatalf("Name = %q, want %q", got, auditweaver.Name)
		}
	})

	t.Run("implements Generator, CapabilityProvider, DirectiveProvider", func(t *testing.T) {
		t.Parallel()
		p := auditweaver.New()
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

	t.Run("Requires the trace capability", func(t *testing.T) {
		t.Parallel()
		got := auditweaver.New().Requires()
		if len(got) != 1 || got[0] != auditweaver.RequiresTrace {
			t.Fatalf("Requires = %+v, want [%q]", got, auditweaver.RequiresTrace)
		}
	})
}

// TestGenerate_WeavesAuditRecord runs repogen + audit-weaver and
// asserts every repogen-emitted method picks up the audit record
// contribution in its rendered body. The auditweaver options are
// pointed at an `audit` import path so the rendered call reads
// `audit.Record(...)` — the canonical demo shape; the renderer
// registers the import on the host file's import set via the
// [emit.NewExternal] expression the plugin emits.
func TestGenerate_WeavesAuditRecord(t *testing.T) {
	t.Parallel()

	result := demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{repogen.New(), auditweaver.New()},
		Backend:       backend_golang.New(),
		Layout:        pipeline.LayoutCentralised,
		OutputPackage: outputPackage,
		PluginOptions: map[string]map[string]string{
			auditweaver.Name: {
				"package": "audit",
				"func":    "Record",
				"format":  "%s",
			},
		},
	})
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}

	body := sinkBody(t, result.Sink, "article"+repogen.FilenameSuffix)
	for _, want := range []string{
		`audit.Record("%s", "ArticleRepo.Get")`,
		`audit.Record("%s", "ArticleRepo.List")`,
		`audit.Record("%s", "ArticleRepo.Save")`,
		`audit.Record("%s", "ArticleRepo.Delete")`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("rendered file missing audit record %q; got:\n%s", want, body)
		}
	}
}

// TestGenerate_OrdersAfterTrace runs both weavers together and
// asserts the rendered Prebody lists the debug entry trace before
// the audit record — the spec-mandated ordering driven by audit's
// Requires: ["trace"] declaration. Both weavers run with their
// configurable options pointed at the demo `audit` / `log`
// packages so the rendered output is stable across runs.
func TestGenerate_OrdersAfterTrace(t *testing.T) {
	t.Parallel()

	result := demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{repogen.New(), auditweaver.New(), debugweaver.New()},
		Backend:       backend_golang.New(),
		Layout:        pipeline.LayoutCentralised,
		OutputPackage: outputPackage,
		PluginOptions: map[string]map[string]string{
			auditweaver.Name: {
				"package": "audit",
				"func":    "Record",
				"format":  "%s",
			},
		},
	})
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}

	body := sinkBody(t, result.Sink, "article"+repogen.FilenameSuffix)
	debugIdx := strings.Index(body, `log.Printf("debug: %s entered", "ArticleRepo.Get")`)
	auditIdx := strings.Index(body, `audit.Record("%s", "ArticleRepo.Get")`)
	if debugIdx < 0 || auditIdx < 0 {
		t.Fatalf("rendered file missing one of the prebody contributions: debug=%d audit=%d", debugIdx, auditIdx)
	}
	if debugIdx >= auditIdx {
		t.Fatalf("debug trace must render before audit record; got debug=%d audit=%d", debugIdx, auditIdx)
	}
}

// TestGenerate_SuppressionByDirective covers the per-method
// opt-out path: an emit method carrying `-gen:audit` does not get
// the audit contribution, while a sibling method without the
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
				{Name: auditweaver.DirectiveName, Negated: true},
			},
		},
	}
	included := &emit.Method{Name: "Audited", Owner: host}
	host.Methods = []*emit.Method{suppressed, included}
	pkg.Structs = []*emit.Struct{host}
	if err := s.Emit().AddPackage(pkg); err != nil {
		t.Fatalf("EmitView.AddPackage: %v", err)
	}
	ctx := &plugin.GeneratorContext{
		Store: s, Reader: store.NewReader(s), Diag: diag.New(),
	}
	if err := auditweaver.New().Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if suppressed.Prebody().Len() != 0 {
		t.Fatalf("suppressed method must not receive an audit contribution; got %d entries", suppressed.Prebody().Len())
	}
	if included.Prebody().Len() != 1 {
		t.Fatalf("included method must receive one audit contribution; got %d entries", included.Prebody().Len())
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

// TestConformance runs the framework's plugin-conformance suite
// against this package's plugin. The suite pins the standard
// framework contracts (stable Name, role-interface compliance,
// deterministic capability ordering, unique directive schema
// names, non-empty Versioned version) so a regression on any
// of them surfaces here before downstream tests trip over it.
func TestConformance(t *testing.T) {
	t.Parallel()
	plugintest.RunSuite(t, auditweaver.New())
}
