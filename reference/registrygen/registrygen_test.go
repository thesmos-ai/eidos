// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package registrygen_test

import (
	"io"
	"io/fs"
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
	"go.thesmos.sh/eidos/reference/registrygen"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// outputPackage is the canonical destination for emitted registry
// init blocks under centralised layout.
const outputPackage = "gen"

// TestPluginShape pins the plugin's public-contract surface.
func TestPluginShape(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the documented identifier", func(t *testing.T) {
		t.Parallel()
		if got := registrygen.New().Name(); got != registrygen.Name {
			t.Fatalf("Name = %q, want %q", got, registrygen.Name)
		}
	})

	t.Run("implements the five required role / capability interfaces", func(t *testing.T) {
		t.Parallel()
		p := registrygen.New()
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
		if _, ok := any(p).(plugin.TemplateProvider); !ok {
			t.Fatalf("plugin must implement plugin.TemplateProvider")
		}
	})

	t.Run("Templates ships a registration.tmpl for golang", func(t *testing.T) {
		t.Parallel()
		tfs, ok := registrygen.New().Templates(registrygen.Language)
		if !ok {
			t.Fatalf("Templates(%q) should return a non-empty filesystem", registrygen.Language)
		}
		body := readTemplate(t, tfs, "registration.tmpl")
		if !strings.Contains(body, `define "registrygen.registration"`) {
			t.Fatalf("registration.tmpl should define the plugin's kind; got:\n%s", body)
		}
	})

	t.Run("Templates returns ok=false for other languages", func(t *testing.T) {
		t.Parallel()
		_, ok := registrygen.New().Templates("rust")
		if ok {
			t.Fatalf("Templates should return ok=false for unsupported languages")
		}
	})

	t.Run("implements plugin.FilenameProvider with the documented suffix", func(t *testing.T) {
		t.Parallel()
		p := registrygen.New()
		fp, ok := any(p).(plugin.FilenameProvider)
		if !ok {
			t.Fatalf("plugin must implement plugin.FilenameProvider")
		}
		if got, want := fp.FilenameSuffix(registrygen.Language), registrygen.FilenameSuffix; got != want {
			t.Fatalf("FilenameSuffix(%q) = %q, want %q", registrygen.Language, got, want)
		}
		if got := fp.FilenameSuffix("rust"); got != "" {
			t.Fatalf("FilenameSuffix(rust) = %q, want empty (plugin ships no rust suffix)", got)
		}
		if registrygen.FilenameSuffix == "" {
			t.Fatalf("FilenameSuffix must be non-empty for a routable-decl-emitting plugin")
		}
	})
}

// TestGenerate_OriginAnchoredSlot pins the routing-layer contract:
// each +gen:register struct produces one origin-anchored slot
// contribution rather than a pre-routed File. The Layout phase
// resolves the contribution's Origin to a Target downstream; the
// plugin itself sets no Target.
func TestGenerate_OriginAnchoredSlot(t *testing.T) {
	t.Parallel()

	t.Run("Generate appends one pending slot per +gen:register struct", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		srcPkg := &node.Package{Name: "blog", Path: "example.com/blog"}
		src := &node.Struct{
			Name:    "Article",
			Package: srcPkg.Path,
			BaseNode: node.BaseNode{
				SourcePos:     position.Pos{File: "blog/article.go", Line: 1},
				DirectiveList: []*directive.Directive{{Name: registrygen.DirectiveName}},
			},
		}
		other := &node.Struct{
			Name:    "Comment",
			Package: srcPkg.Path,
			BaseNode: node.BaseNode{
				SourcePos: position.Pos{File: "blog/comment.go", Line: 1},
			},
		}
		srcPkg.Structs = []*node.Struct{src, other}
		if err := s.Nodes().AddPackage(srcPkg); err != nil {
			t.Fatalf("AddPackage: %v", err)
		}
		p := registrygen.New()
		if err := p.SetOptions(opt.New(p.OptionsSchema(), nil)); err != nil {
			t.Fatalf("SetOptions: %v", err)
		}
		ctx := &plugin.GeneratorContext{
			Store: s, Reader: store.NewReader(s), Diag: diag.New(),
		}
		if err := p.Generate(ctx); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		// The +gen:register struct produces one pending slot
		// anchored to itself; the un-annotated sibling produces
		// nothing. No emit.File is created — Layout composes the
		// resolved Target from the origin downstream.
		pending := s.Emit().PendingOriginSlots()
		if len(pending) != 1 {
			t.Fatalf("expected 1 pending origin slot; got %d", len(pending))
		}
		tup := pending[0]
		if tup.Origin != src {
			t.Fatalf("pending tuple Origin = %+v, want the +gen:register struct", tup.Origin)
		}
		if tup.SlotName != "init" {
			t.Fatalf("slot name = %q, want %q", tup.SlotName, "init")
		}
		if _, ok := tup.Item.(*registrygen.Registration); !ok {
			t.Fatalf("slot item should be *Registration; got %T", tup.Item)
		}
		// No File pre-routed by the plugin.
		if files := s.Emit().Files().Len(); files != 0 {
			t.Fatalf("plugin should not pre-route any emit.File; got %d", files)
		}
	})
}

// TestRegistrationKind covers the plugin-defined emit kind: a
// Registration reports its own kind so the backend's template
// dispatcher routes it through the plugin's registration.tmpl
// rather than the core `emit.*` templates.
func TestRegistrationKind(t *testing.T) {
	t.Parallel()

	r := &registrygen.Registration{Name: "Probe"}
	if got, want := string(r.Kind()), string(registrygen.Kind); got != want {
		t.Fatalf("Registration.Kind() = %q, want %q", got, want)
	}
}

// TestGenerate_EndToEnd runs registry-gen against the demoproject
// fixture and asserts the canonical acceptance criteria for the
// origin-anchored slot path: Article (the only fixture struct
// carrying +gen:register) produces a Registration anchored to
// its source struct; the Layout phase routes the contribution
// into a per-source registry file (`article_registry.go` under
// the centralised package); and the rendered file contains the
// expected register call inside a single `func init()` block.
func TestGenerate_EndToEnd(t *testing.T) {
	t.Parallel()

	result := demopipe.Run(t, demopipe.RunOptions{
		Generators:    []plugin.Generator{registrygen.New()},
		Backend:       backend_golang.New(),
		Layout:        pipeline.LayoutCentralised,
		OutputPackage: outputPackage,
		PluginOptions: map[string]map[string]string{
			registrygen.Name: {
				"register_package": "registry",
				"register_func":    "Register",
			},
		},
	})
	if result.Diag.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
	}
	if result.RunErr != nil {
		t.Fatalf("pipeline Run: %v", result.RunErr)
	}

	t.Run("Article registration lands in the per-source registry file", func(t *testing.T) {
		t.Parallel()
		file := requireArticleRegistryFile(t, result.Store)
		entries := file.Init().Items
		if len(entries) != 1 {
			t.Fatalf("expected 1 Init entry; got %d", len(entries))
		}
		reg, ok := entries[0].(*registrygen.Registration)
		if !ok {
			t.Fatalf("Init entry should be *Registration; got %T", entries[0])
		}
		if reg.Name != "Article" {
			t.Fatalf("Registration.Name = %q, want %q", reg.Name, "Article")
		}
	})

	t.Run("rendered file contains the registry.Register call inside one func init() block", func(t *testing.T) {
		t.Parallel()
		body := sinkBody(t, result.Sink, "article"+registrygen.FilenameSuffix)
		if want := `registry.Register("Article", blog.Article{})`; !strings.Contains(body, want) {
			t.Fatalf("rendered file missing %q; got:\n%s", want, body)
		}
		if strings.Count(body, "func init()") != 1 {
			t.Fatalf("expected exactly one func init() block; got:\n%s", body)
		}
	})

	t.Run("non-+gen:register fixture structs do not produce registrations", func(t *testing.T) {
		t.Parallel()
		// The demoproject contains many structs; only Article
		// carries +gen:register. Pending slot count therefore
		// stays at 1 — no other registration was queued.
		if got := len(result.Store.Emit().PendingOriginSlots()); got != 1 {
			t.Fatalf("PendingOriginSlots = %d, want 1 (Article only)", got)
		}
	})
}

// readTemplate pulls the named template body out of fsys as a
// string, failing the test if the entry is missing.
func readTemplate(t *testing.T, fsys fs.FS, name string) string {
	t.Helper()
	f, err := fsys.Open(name)
	if err != nil {
		t.Fatalf("open %q: %v", name, err)
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read %q: %v", name, err)
	}
	return string(body)
}

// requireArticleRegistryFile returns the emit.File composed by
// the Layout phase for the Article registration (basename
// "article" + the plugin's filename suffix). Fails the test when
// the file is missing.
func requireArticleRegistryFile(t *testing.T, s *store.Store) *emit.File {
	t.Helper()
	want := "article" + registrygen.FilenameSuffix
	for _, f := range s.Emit().Files().Items() {
		if f.Name == want {
			return f
		}
	}
	t.Fatalf("emit store missing file %q", want)
	return nil
}

// sinkBody returns the rendered body for the file whose basename
// equals filename. The Layout phase composes the file under the
// configured centralised output package; the helper finds it via
// the filename portion alone.
func sinkBody(t *testing.T, s sink.Sink, filename string) string {
	t.Helper()
	mem, ok := s.(*sink.Memory)
	if !ok {
		t.Fatalf("sink is not *sink.Memory; got %T", s)
	}
	for target, body := range mem.Files() {
		if target.Filename == filename {
			return string(body)
		}
	}
	t.Fatalf("sink missing file %q", filename)
	return ""
}
