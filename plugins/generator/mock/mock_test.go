// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package mock_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/plugins/generator/mock"
	"go.thesmos.sh/eidos/priority"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/store"
)

// TestPlugin_Contract pins the framework contract surface: stable
// name, role compliance, capability metadata, and graceful empty-
// store behaviour.
func TestPlugin_Contract(t *testing.T) {
	t.Parallel()

	t.Run("Name is the package-exported constant", func(t *testing.T) {
		t.Parallel()
		p := mock.New()
		if got := p.Name(); got != mock.Name {
			t.Fatalf("Name() = %q, want %q", got, mock.Name)
		}
	})

	t.Run("Priority is GeneratorComposition", func(t *testing.T) {
		t.Parallel()
		if got, want := mock.New().Priority(), priority.GeneratorComposition; got != want {
			t.Fatalf("Priority() = %v, want %v", got, want)
		}
	})

	t.Run("Provides advertises the mock capability", func(t *testing.T) {
		t.Parallel()
		got := mock.New().Provides()
		if len(got) != 1 || got[0] != mock.Capability {
			t.Fatalf("Provides() = %v, want [%q]", got, mock.Capability)
		}
	})

	t.Run("satisfies plugin.Generator", func(t *testing.T) {
		t.Parallel()
		var _ plugin.Generator = mock.New()
	})

	t.Run("Directives declares +gen:mock", func(t *testing.T) {
		t.Parallel()
		schemas := mock.New().Directives()
		if len(schemas) != 1 || schemas[0].Name != mock.DirectiveName {
			t.Fatalf("Directives() = %v, want one %q schema", schemas, mock.DirectiveName)
		}
	})

	t.Run("Generate on empty store is a no-op", func(t *testing.T) {
		t.Parallel()
		ctx := newGeneratorContext(t, store.New())
		if err := mock.New().Generate(ctx); err != nil {
			t.Fatalf("Generate(empty): %v", err)
		}
	})
}

// TestPlugin_EmitsMockStruct covers the canonical happy path —
// one interface annotated with +gen:mock yields a <Type>Mock
// struct with one On<Method> field plus the dispatch method per
// interface method.
func TestPlugin_EmitsMockStruct(t *testing.T) {
	t.Parallel()

	iface := &node.Interface{
		Name: "Store", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{
				{Name: mock.DirectiveName},
			},
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
	ctx := contextWithInterface(t, iface)
	if err := mock.New().Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	got := findStruct(ctx.Store.Emit(), "x.StoreMock")
	if got == nil {
		t.Fatalf("StoreMock struct not emitted; got buckets = %d structs",
			ctx.Store.Emit().Structs().Len())
	}

	t.Run("has OnGet field with func type", func(t *testing.T) {
		t.Parallel()
		var found bool
		for _, f := range got.Fields {
			if f.Name == "OnGet" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("OnGet field missing on StoreMock; fields = %v", fieldNames(got))
		}
	})

	t.Run("has Get dispatch method", func(t *testing.T) {
		t.Parallel()
		var m *emit.Method
		for _, mm := range got.Methods {
			if mm.Name == "Get" {
				m = mm
				break
			}
		}
		if m == nil {
			t.Fatalf("Get method missing on StoreMock")
		}
		if len(m.Body) == 0 {
			t.Fatalf("Get method has empty body")
		}
	})

	t.Run("stamps mock.iface with source qname", func(t *testing.T) {
		t.Parallel()
		gotQName, ok := mock.MetaIface.Get(got.Meta())
		if !ok {
			t.Fatalf("MetaIface not stamped on emitted struct")
		}
		if gotQName != "x.Store" {
			t.Fatalf("MetaIface = %q, want %q", gotQName, "x.Store")
		}
	})
}

// TestPlugin_SlotsAreAccessible exercises the documented
// extension surface — downstream plugins write to the standard
// slots on the emitted struct + method without coordinating with
// the mock plugin.
func TestPlugin_SlotsAreAccessible(t *testing.T) {
	t.Parallel()

	iface := &node.Interface{
		Name: "Store", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{{Name: mock.DirectiveName}},
		},
		Methods: []*node.Method{
			{Name: "Get", Returns: []*node.TypeRef{{Name: "error"}}},
		},
	}
	ctx := contextWithInterface(t, iface)
	if err := mock.New().Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	s := findStruct(ctx.Store.Emit(), "x.StoreMock")
	if s == nil {
		t.Fatalf("StoreMock not emitted")
	}

	t.Run("Struct fields slot accepts cross-cutting field appends", func(t *testing.T) {
		t.Parallel()
		extra := &emit.Field{Name: "calls", Type: emit.Builtin("int")}
		if err := s.FieldsSlot().Append(extra, emit.Provenance{SetBy: "downstream"}); err != nil {
			t.Fatalf("FieldsSlot append: %v", err)
		}
		if s.FieldsSlot().Len() != 1 {
			t.Fatalf("FieldsSlot.Len() = %d, want 1", s.FieldsSlot().Len())
		}
	})

	t.Run("Method prebody slot accepts cross-cutting stmt appends", func(t *testing.T) {
		t.Parallel()
		var m *emit.Method
		for _, mm := range s.Methods {
			if mm.Name == "Get" {
				m = mm
				break
			}
		}
		if m == nil {
			t.Fatalf("Get method missing")
		}
		stmt := emit.NewExprStmt(emit.NewCall(emit.NewIdent("record")))
		if err := m.Prebody().Append(stmt, emit.Provenance{SetBy: "downstream"}); err != nil {
			t.Fatalf("Prebody append: %v", err)
		}
		if m.Prebody().Len() != 1 {
			t.Fatalf("Prebody.Len() = %d, want 1", m.Prebody().Len())
		}
	})
}

// TestPlugin_VoidMethod_NoTrailingNakedReturn pins the dispatch
// body shape for void methods: when the source method declares no
// returns, the body is the single nil-check if-block — there is
// no zero-value fall-through to yield, so a trailing naked
// `return` would be redundant noise in the generated source.
func TestPlugin_VoidMethod_NoTrailingNakedReturn(t *testing.T) {
	t.Parallel()

	iface := &node.Interface{
		Name: "Notifier", Package: "x",
		BaseNode: node.BaseNode{
			DirectiveList: []*directive.Directive{{Name: mock.DirectiveName}},
		},
		Methods: []*node.Method{
			{Name: "Notify"},
		},
	}
	ctx := contextWithInterface(t, iface)
	if err := mock.New().Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	got := findStruct(ctx.Store.Emit(), "x.NotifierMock")
	if got == nil {
		t.Fatalf("NotifierMock struct not emitted")
	}
	var m *emit.Method
	for _, mm := range got.Methods {
		if mm.Name == "Notify" {
			m = mm
			break
		}
	}
	if m == nil {
		t.Fatalf("Notify method missing on NotifierMock")
	}

	t.Run("body contains exactly one statement", func(t *testing.T) {
		t.Parallel()
		if len(m.Body) != 1 {
			t.Fatalf("void dispatch body length = %d, want 1", len(m.Body))
		}
	})

	t.Run("the lone statement is the nil-check if-block", func(t *testing.T) {
		t.Parallel()
		if len(m.Body) == 0 {
			t.Fatalf("void dispatch body is empty")
		}
		if m.Body[0].StmtKind != emit.StmtIf {
			t.Fatalf("void dispatch body[0] = %v, want StmtIf", m.Body[0].StmtKind)
		}
	})
}

// TestPlugin_SkipsUnannotated covers the directive gate —
// interfaces without +gen:mock are not mocked, including the
// negated form.
func TestPlugin_SkipsUnannotated(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		iface *node.Interface
	}{
		{
			name: "no directive",
			iface: &node.Interface{
				Name: "Store", Package: "x",
				Methods: []*node.Method{{Name: "Get"}},
			},
		},
		{
			name: "negated directive",
			iface: &node.Interface{
				Name: "Store", Package: "x",
				BaseNode: node.BaseNode{
					DirectiveList: []*directive.Directive{
						{Name: mock.DirectiveName, Negated: true},
					},
				},
				Methods: []*node.Method{{Name: "Get"}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := contextWithInterface(t, tc.iface)
			if err := mock.New().Generate(ctx); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if ctx.Store.Emit().Structs().Len() != 0 {
				t.Fatalf("expected no emit; got %d structs",
					ctx.Store.Emit().Structs().Len())
			}
		})
	}
}

// contextWithInterface builds a GeneratorContext seeded with a
// single-interface package — the canonical setup for every
// per-interface test in this file.
func contextWithInterface(t *testing.T, i *node.Interface) *sdk.GeneratorContext {
	t.Helper()
	s := store.New()
	if err := s.Nodes().AddPackage(&node.Package{
		Name: "x", Path: "x",
		Interfaces: []*node.Interface{i},
	}); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}
	return newGeneratorContext(t, s)
}

// newGeneratorContext wraps s in a fresh GeneratorContext with
// read-tracking reader and an empty diagnostic sink.
func newGeneratorContext(t *testing.T, s *store.Store) *sdk.GeneratorContext {
	t.Helper()
	return &sdk.GeneratorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   diag.New(),
	}
}

// findStruct returns the emit struct registered under qname, or
// nil when no such struct exists. The mock plugin emits at
// <pkg>.<Type>Mock so callers pass that qname directly.
func findStruct(v *store.EmitView, qname string) *emit.Struct {
	got, _ := v.Structs().ByQName(qname)
	return got
}

// fieldNames returns the names of every field on s for failure
// messages that need to enumerate the actual emitted state.
func fieldNames(s *emit.Struct) []string {
	out := make([]string, 0, len(s.Fields))
	for _, f := range s.Fields {
		out = append(out, f.Name)
	}
	return out
}
