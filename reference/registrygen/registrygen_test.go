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
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/registrygen"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// outputPackage is the canonical destination for emitted registry
// init blocks.
const outputPackage = "gen"

// defaultFilename mirrors the plugin's Options.Filename default so
// tests look up the right Target.
const defaultFilename = "registry.go"

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
// fixture and asserts the canonical acceptance criteria:
//   - Article (the only fixture struct carrying +gen:register)
//     produces a Registration in the target file's Init slot;
//   - the rendered file contains one registry.Register call inside
//     one func init() block; and
//   - the plugin's template reached the backend (the rendered
//     output matches the template's shape).
func TestGenerate_EndToEnd(t *testing.T) {
	t.Parallel()

	result := demopipe.Run(t, demopipe.RunOptions{
		Generators: []plugin.Generator{registrygen.New()},
		Backend:    backend_golang.New(),
		PluginOptions: map[string]map[string]string{
			registrygen.Name: {
				"output_package":   outputPackage,
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

	t.Run("Article registration lands in the emit-store File.Init slot", func(t *testing.T) {
		t.Parallel()
		file := requireFile(t, result.Store)
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
		body := sinkBody(t, result.Sink, defaultFilename)
		if want := `registry.Register("Article", blog.Article{})`; !strings.Contains(body, want) {
			t.Fatalf("registry.go missing %q; got:\n%s", want, body)
		}
		if strings.Count(body, "func init()") != 1 {
			t.Fatalf("expected exactly one func init() block; got:\n%s", body)
		}
	})

	t.Run("non-+gen:register fixture structs do not produce registrations", func(t *testing.T) {
		t.Parallel()
		file := requireFile(t, result.Store)
		if file.Init().Len() != 1 {
			t.Fatalf("Init slot should hold exactly the Article entry; got %d items", file.Init().Len())
		}
	})
}

// TestGenerate_AlongsideSourceLayout covers the default layout:
// an unconfigured plugin emits one `registry.go` per source
// package, with [emit.Target.Dir] derived from the package's first
// source file. The init block lives alongside the package it
// registers.
func TestGenerate_AlongsideSourceLayout(t *testing.T) {
	t.Parallel()

	t.Run("registry.go drops alongside the source package", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		srcPkg := &node.Package{
			Name: "blog", Path: "example.com/blog",
			Files: []*node.File{{Path: "blog/article.go"}},
		}
		src := &node.Struct{
			Name:    "Article",
			Package: srcPkg.Path,
			BaseNode: node.BaseNode{
				SourcePos:     position.Pos{File: "blog/article.go", Line: 1},
				DirectiveList: []*directive.Directive{{Name: registrygen.DirectiveName}},
			},
		}
		srcPkg.Structs = append(srcPkg.Structs, src)
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

		target := emit.Target{Dir: "blog", Filename: registrygen.DefaultFilename, Package: "blog"}
		file, err := s.Emit().FileFor(target)
		if err != nil {
			t.Fatalf("FileFor: %v", err)
		}
		entries := file.Init().Items
		if len(entries) != 1 {
			t.Fatalf("expected 1 Init entry alongside source; got %d", len(entries))
		}
		reg, ok := entries[0].(*registrygen.Registration)
		if !ok {
			t.Fatalf("Init entry should be *Registration; got %T", entries[0])
		}
		if reg.Name != "Article" {
			t.Fatalf("Registration.Name = %q, want Article", reg.Name)
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

// requireFile returns the registry-gen target file from the emit
// store, failing the test when the file is missing.
func requireFile(t *testing.T, s *store.Store) *emit.File {
	t.Helper()
	target := emit.Target{Dir: outputPackage, Filename: defaultFilename, Package: outputPackage}
	for _, f := range s.Emit().Files().Items() {
		if f.Target() == target {
			return f
		}
	}
	t.Fatalf("emit store missing the registry-gen target file %+v", target)
	return nil
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
