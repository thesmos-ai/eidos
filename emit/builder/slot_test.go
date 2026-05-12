// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
)

// TestAppendField covers the canonical AppendField path: provenance
// is stamped with the Context's plugin name and the host's Owner
// back-pointer is wired.
func TestAppendField(t *testing.T) {
	t.Parallel()

	t.Run("appends field with Provenance.SetBy stamped", func(t *testing.T) {
		t.Parallel()
		c := builder.For("validation", defaultTarget)
		host := &emit.Struct{Name: "User", Package: "users", Target: defaultTarget}
		err := c.AppendField(host, &emit.Field{Name: "Email", Type: emit.Builtin("string"), Tag: `validate:"email"`})
		if err != nil {
			t.Fatalf("AppendField returned %v", err)
		}
		slot := host.FieldsSlot()
		if slot.Len() != 1 {
			t.Fatalf("expected 1 contribution; got %d", slot.Len())
		}
		f, ok := slot.At(0).(*emit.Field)
		if !ok || f.Name != "Email" {
			t.Fatalf("contribution shape wrong; got %+v", slot.At(0))
		}
		if f.Owner != host {
			t.Fatalf("Owner not wired; got %v, want %p", f.Owner, host)
		}
		if prov := slot.ProvenanceAt(0); prov.SetBy != "validation" {
			t.Fatalf("Provenance.SetBy = %q, want %q", prov.SetBy, "validation")
		}
	})

	t.Run("optional id is recorded in Provenance.ID", func(t *testing.T) {
		t.Parallel()
		c := builder.For("validation", defaultTarget)
		host := &emit.Struct{Name: "User", Package: "users", Target: defaultTarget}
		_ = c.AppendField(host, &emit.Field{Name: "Email", Type: emit.Builtin("string")}, "validation/email")
		if prov := host.FieldsSlot().ProvenanceAt(0); prov.ID != "validation/email" {
			t.Fatalf("Provenance.ID = %q, want validation/email", prov.ID)
		}
	})

	t.Run("nil host returns ErrNilHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("validation", defaultTarget)
		err := c.AppendField(nil, &emit.Field{Name: "X", Type: emit.Builtin("int")})
		if !errors.Is(err, builder.ErrNilHost) {
			t.Fatalf("expected ErrNilHost; got %v", err)
		}
	})

	t.Run("non-struct host returns ErrUnsupportedHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("validation", defaultTarget)
		err := c.AppendField(&emit.Interface{Name: "I"}, &emit.Field{Name: "X", Type: emit.Builtin("int")})
		if !errors.Is(err, builder.ErrUnsupportedHost) {
			t.Fatalf("expected ErrUnsupportedHost; got %v", err)
		}
	})
}

// TestAppendMethod covers method appends across all three accepted
// host kinds (Struct, Interface, Alias) plus error paths.
func TestAppendMethod(t *testing.T) {
	t.Parallel()

	t.Run("appends to *emit.Struct, *emit.Interface, and *emit.Alias", func(t *testing.T) {
		t.Parallel()
		c := builder.For("mockgen", defaultTarget)

		s := &emit.Struct{Name: "Repo"}
		if err := c.AppendMethod(s, &emit.Method{Name: "Get"}); err != nil {
			t.Fatalf("AppendMethod(struct) = %v", err)
		}
		if s.MethodsSlot().Len() != 1 {
			t.Fatalf("struct methods slot did not receive contribution")
		}

		i := &emit.Interface{Name: "Repo"}
		if err := c.AppendMethod(i, &emit.Method{Name: "Get"}); err != nil {
			t.Fatalf("AppendMethod(interface) = %v", err)
		}
		if i.MethodsSlot().Len() != 1 {
			t.Fatalf("interface methods slot did not receive contribution")
		}

		a := &emit.Alias{Name: "UserID"}
		if err := c.AppendMethod(a, &emit.Method{Name: "Validate"}); err != nil {
			t.Fatalf("AppendMethod(alias) = %v", err)
		}
		if a.MethodsSlot().Len() != 1 {
			t.Fatalf("alias methods slot did not receive contribution")
		}
	})

	t.Run("unsupported host kind returns ErrUnsupportedHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("mockgen", defaultTarget)
		err := c.AppendMethod(&emit.Function{Name: "F"}, &emit.Method{Name: "M"})
		if !errors.Is(err, builder.ErrUnsupportedHost) {
			t.Fatalf("expected ErrUnsupportedHost; got %v", err)
		}
	})
}

// TestAppendVariant covers EnumVariant slot appends — the only
// host kind is *emit.Enum, and provenance threads through.
func TestAppendVariant(t *testing.T) {
	t.Parallel()

	t.Run("appends variant with Owner and Provenance wired", func(t *testing.T) {
		t.Parallel()
		c := builder.For("statusgen", defaultTarget)
		e := &emit.Enum{Name: "State"}
		err := c.AppendVariant(e, &emit.EnumVariant{Name: "Pending"})
		if err != nil {
			t.Fatalf("AppendVariant = %v", err)
		}
		v, ok := e.VariantsSlot().At(0).(*emit.EnumVariant)
		if !ok || v.Name != "Pending" {
			t.Fatalf("contribution shape wrong; got %+v", e.VariantsSlot().At(0))
		}
		if v.Owner != e {
			t.Fatalf("Owner not wired; got %v, want %p", v.Owner, e)
		}
		if prov := e.VariantsSlot().ProvenanceAt(0); prov.SetBy != "statusgen" {
			t.Fatalf("Provenance.SetBy = %q, want statusgen", prov.SetBy)
		}
	})

	t.Run("nil host returns ErrNilHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("statusgen", defaultTarget)
		if err := c.AppendVariant(nil, &emit.EnumVariant{Name: "X"}); !errors.Is(err, builder.ErrNilHost) {
			t.Fatalf("expected ErrNilHost; got %v", err)
		}
	})
}

// TestAppendPrebody covers prebody contributions on both Function
// and Method hosts.
func TestAppendPrebody(t *testing.T) {
	t.Parallel()

	t.Run("appends Stmt on Function and Method", func(t *testing.T) {
		t.Parallel()
		c := builder.For("metricsgen", defaultTarget)
		stmt := emit.NewRawStmt(`start := time.Now()`)

		f := &emit.Function{Name: "F"}
		if err := c.AppendPrebody(f, stmt); err != nil {
			t.Fatalf("AppendPrebody(function) = %v", err)
		}
		if f.Prebody().Len() != 1 {
			t.Fatalf("function prebody empty after append")
		}

		m := &emit.Method{Name: "M"}
		if err := c.AppendPrebody(m, stmt); err != nil {
			t.Fatalf("AppendPrebody(method) = %v", err)
		}
		if m.Prebody().Len() != 1 {
			t.Fatalf("method prebody empty after append")
		}
	})

	t.Run("unsupported host returns ErrUnsupportedHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("metricsgen", defaultTarget)
		err := c.AppendPrebody(&emit.Struct{Name: "S"}, emit.NewRawStmt(""))
		if !errors.Is(err, builder.ErrUnsupportedHost) {
			t.Fatalf("expected ErrUnsupportedHost; got %v", err)
		}
	})
}

// TestAppendFileSlots covers the File-level slot appends (Top,
// Bottom, Init, Import) all go through the same provenance path.
func TestAppendFileSlots(t *testing.T) {
	t.Parallel()

	t.Run("Top / Bottom / Init / Import each carry SetBy", func(t *testing.T) {
		t.Parallel()
		c := builder.For("audit", defaultTarget)
		file := &emit.File{Name: "users.go", Package: "users", Dir: "users"}

		topDecl := &emit.Constant{Name: "K"}
		if err := c.AppendTop(file, topDecl); err != nil {
			t.Fatalf("AppendTop = %v", err)
		}
		bottomDecl := &emit.Constant{Name: "K2"}
		if err := c.AppendBottom(file, bottomDecl); err != nil {
			t.Fatalf("AppendBottom = %v", err)
		}
		if err := c.AppendInit(file, emit.NewRawStmt(`audit.Register()`)); err != nil {
			t.Fatalf("AppendInit = %v", err)
		}
		if err := c.AppendImport(file, &emit.Import{Path: "go.example.com/audit"}); err != nil {
			t.Fatalf("AppendImport = %v", err)
		}

		for _, s := range []struct {
			name string
			prov emit.Provenance
		}{
			{"top", file.Top().ProvenanceAt(0)},
			{"bottom", file.Bottom().ProvenanceAt(0)},
			{"init", file.Init().ProvenanceAt(0)},
			{"imports", file.ImportsSlot().ProvenanceAt(0)},
		} {
			if s.prov.SetBy != "audit" {
				t.Fatalf("%s slot SetBy = %q, want audit", s.name, s.prov.SetBy)
			}
		}
	})

	t.Run("nil file returns ErrNilHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("audit", defaultTarget)
		if err := c.AppendInit(nil, emit.NewRawStmt("")); !errors.Is(err, builder.ErrNilHost) {
			t.Fatalf("expected ErrNilHost; got %v", err)
		}
	})
}

// TestAppendEmbed_AcceptedHosts covers AppendEmbed across the two
// accepted host kinds (Struct, Interface) plus the unsupported-host
// rejection path.
func TestAppendEmbed_AcceptedHosts(t *testing.T) {
	t.Parallel()

	t.Run("appends Embed on Struct and Interface; Owner wired", func(t *testing.T) {
		t.Parallel()
		c := builder.For("audit", defaultTarget)
		s := &emit.Struct{Name: "S"}
		if err := c.AppendEmbed(s, &emit.Embed{Type: emit.Builtin("Reader")}); err != nil {
			t.Fatalf("AppendEmbed(struct) = %v", err)
		}
		if e := s.EmbedsSlot().At(0).(*emit.Embed); e.Owner != s {
			t.Fatalf("embed Owner not wired on struct host")
		}

		i := &emit.Interface{Name: "I"}
		if err := c.AppendEmbed(i, &emit.Embed{Type: emit.Builtin("Reader")}); err != nil {
			t.Fatalf("AppendEmbed(interface) = %v", err)
		}
		if e := i.EmbedsSlot().At(0).(*emit.Embed); e.Owner != i {
			t.Fatalf("embed Owner not wired on interface host")
		}
	})

	t.Run("unsupported host returns ErrUnsupportedHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("audit", defaultTarget)
		err := c.AppendEmbed(&emit.Function{Name: "F"}, &emit.Embed{})
		if !errors.Is(err, builder.ErrUnsupportedHost) {
			t.Fatalf("expected ErrUnsupportedHost; got %v", err)
		}
	})

	t.Run("nil host returns ErrNilHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("audit", defaultTarget)
		if err := c.AppendEmbed(nil, &emit.Embed{}); !errors.Is(err, builder.ErrNilHost) {
			t.Fatalf("expected ErrNilHost; got %v", err)
		}
	})
}

// TestAppendParamAndPostbody covers AppendParam and AppendPostbody
// — the same Function / Method host pattern as AppendPrebody, both
// previously uncovered by the canonical-path test.
func TestAppendParamAndPostbody(t *testing.T) {
	t.Parallel()

	t.Run("AppendParam wires Owner on Function and Method", func(t *testing.T) {
		t.Parallel()
		c := builder.For("auth", defaultTarget)
		f := &emit.Function{Name: "F"}
		p := &emit.Param{Name: "ctx", Type: emit.External("context", "Context")}
		if err := c.AppendParam(f, p); err != nil {
			t.Fatalf("AppendParam(function) = %v", err)
		}
		if p.Owner != f {
			t.Fatalf("param Owner not wired on function")
		}
		m := &emit.Method{Name: "M"}
		p2 := &emit.Param{Name: "ctx", Type: emit.External("context", "Context")}
		if err := c.AppendParam(m, p2); err != nil {
			t.Fatalf("AppendParam(method) = %v", err)
		}
		if p2.Owner != m {
			t.Fatalf("param Owner not wired on method")
		}
	})

	t.Run("AppendParam rejects unsupported host", func(t *testing.T) {
		t.Parallel()
		c := builder.For("auth", defaultTarget)
		if err := c.AppendParam(&emit.Struct{}, &emit.Param{}); !errors.Is(err, builder.ErrUnsupportedHost) {
			t.Fatalf("expected ErrUnsupportedHost; got %v", err)
		}
	})

	t.Run("AppendPostbody appends Stmt on Function and Method", func(t *testing.T) {
		t.Parallel()
		c := builder.For("metricsgen", defaultTarget)
		stmt := emit.NewRawStmt(`metrics.Report()`)
		f := &emit.Function{Name: "F"}
		if err := c.AppendPostbody(f, stmt); err != nil {
			t.Fatalf("AppendPostbody(function) = %v", err)
		}
		if f.Postbody().Len() != 1 {
			t.Fatalf("function postbody empty after append")
		}
		m := &emit.Method{Name: "M"}
		if err := c.AppendPostbody(m, stmt); err != nil {
			t.Fatalf("AppendPostbody(method) = %v", err)
		}
		if m.Postbody().Len() != 1 {
			t.Fatalf("method postbody empty after append")
		}
	})

	t.Run("AppendPostbody rejects unsupported host", func(t *testing.T) {
		t.Parallel()
		c := builder.For("metricsgen", defaultTarget)
		if err := c.AppendPostbody(&emit.Struct{}, emit.NewRawStmt("")); !errors.Is(err, builder.ErrUnsupportedHost) {
			t.Fatalf("expected ErrUnsupportedHost; got %v", err)
		}
	})
}

// TestSlotHelpers_NilHostGuards covers the ErrNilHost arms of the
// slot helpers whose canonical-path tests skip them. Centralised so
// the guard discipline is asserted in one place rather than
// scattered across per-helper tests.
func TestSlotHelpers_NilHostGuards(t *testing.T) {
	t.Parallel()

	c := builder.For("test", defaultTarget)
	cases := []struct {
		name string
		fn   func() error
	}{
		{"AppendMethod", func() error { return c.AppendMethod(nil, &emit.Method{}) }},
		{"AppendParam", func() error { return c.AppendParam(nil, &emit.Param{}) }},
		{"AppendPrebody", func() error { return c.AppendPrebody(nil, emit.NewRawStmt("")) }},
		{"AppendPostbody", func() error { return c.AppendPostbody(nil, emit.NewRawStmt("")) }},
	}
	for _, tc := range cases {
		t.Run(tc.name+" rejects nil host", func(t *testing.T) {
			t.Parallel()
			if err := tc.fn(); !errors.Is(err, builder.ErrNilHost) {
				t.Fatalf("expected ErrNilHost; got %v", err)
			}
		})
	}
}

// TestAppendFileSlots_NilGuards covers the ErrNilHost arms of
// AppendTop, AppendBottom, and AppendImport — the canonical-path
// test only exercises the happy path.
func TestAppendFileSlots_NilGuards(t *testing.T) {
	t.Parallel()

	t.Run("nil file returns ErrNilHost for Top/Bottom/Import", func(t *testing.T) {
		t.Parallel()
		c := builder.For("audit", defaultTarget)
		if err := c.AppendTop(nil, &emit.Constant{}); !errors.Is(err, builder.ErrNilHost) {
			t.Fatalf("AppendTop: expected ErrNilHost; got %v", err)
		}
		if err := c.AppendBottom(nil, &emit.Constant{}); !errors.Is(err, builder.ErrNilHost) {
			t.Fatalf("AppendBottom: expected ErrNilHost; got %v", err)
		}
		if err := c.AppendImport(nil, &emit.Import{Path: "x"}); !errors.Is(err, builder.ErrNilHost) {
			t.Fatalf("AppendImport: expected ErrNilHost; got %v", err)
		}
	})
}

// TestNodeAccessors covers the Node() accessors on every sub-element
// builder — needed for callers that want to capture the pointer
// for later cross-cutting reference.
func TestNodeAccessors(t *testing.T) {
	t.Parallel()

	t.Run("each sub-element builder's Node() returns the underlying pointer", func(t *testing.T) {
		t.Parallel()
		c := builder.For("test", defaultTarget)
		pkg := c.Package("p", "p")
		if pkg.Node() == nil {
			t.Fatalf("PackageBuilder.Node returned nil")
		}
		pkg.
			Struct("S", func(sb *builder.StructBuilder) {
				sb.Field("F", emit.Builtin("int"), func(fb *builder.FieldBuilder) {
					if fb.Node() == nil {
						t.Fatalf("FieldBuilder.Node returned nil")
					}
				})
				sb.Embed(emit.Builtin("Reader"), func(eb *builder.EmbedBuilder) {
					if eb.Node() == nil {
						t.Fatalf("EmbedBuilder.Node returned nil")
					}
				})
			}).
			Enum("E", emit.Builtin("int"), func(eb *builder.EnumBuilder) {
				eb.Variant("V", nil, func(vb *builder.EnumVariantBuilder) {
					if vb.Node() == nil {
						t.Fatalf("EnumVariantBuilder.Node returned nil")
					}
				})
			}).
			File(emit.Target{}, func(fb *builder.FileBuilder) {
				fb.Import("fmt", func(ib *builder.ImportBuilder) {
					if ib.Node() == nil {
						t.Fatalf("ImportBuilder.Node returned nil")
					}
				})
			})
	})
}

// TestAppendTag covers the field-tag slot append — a less common
// host (the host is a *emit.Field, not a decl) but exercised the
// same Provenance discipline.
func TestAppendTag(t *testing.T) {
	t.Parallel()

	t.Run("appends Tag to the field's tags slot with SetBy", func(t *testing.T) {
		t.Parallel()
		c := builder.For("jsongen", defaultTarget)
		f := &emit.Field{Name: "Email", Type: emit.Builtin("string")}
		if err := c.AppendTag(f, &emit.Tag{Key: "json", Value: "email,omitempty"}); err != nil {
			t.Fatalf("AppendTag = %v", err)
		}
		if got := f.Tags().Len(); got != 1 {
			t.Fatalf("tags slot len = %d, want 1", got)
		}
		if prov := f.Tags().ProvenanceAt(0); prov.SetBy != "jsongen" {
			t.Fatalf("Provenance.SetBy = %q, want jsongen", prov.SetBy)
		}
	})

	t.Run("nil host returns ErrNilHost", func(t *testing.T) {
		t.Parallel()
		c := builder.For("jsongen", defaultTarget)
		if err := c.AppendTag(nil, &emit.Tag{Key: "json"}); !errors.Is(err, builder.ErrNilHost) {
			t.Fatalf("expected ErrNilHost; got %v", err)
		}
	})
}
