// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package golang_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/eidostest/testpipe"
	"go.thesmos.sh/eidos/emit"
)

// TestStructEmbeds_Render covers struct embedding via the
// emit.struct template's renderEmbeds invocation.
func TestStructEmbeds_Render(t *testing.T) {
	t.Parallel()

	t.Run("struct with single embed renders the embedded type", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Structs: []*emit.Struct{{
				Name: "Wrapper", Package: "x", Target: target,
				Embeds: []*emit.Embed{
					{Type: emit.External("io", "Reader")},
				},
				Fields: []*emit.Field{{Name: "Closed", Type: emit.Builtin("bool")}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "io.Reader\n\tClosed bool") {
			t.Fatalf("embed should precede fields; got:\n%s", body)
		}
	})

	t.Run("struct with multiple embeds renders each on its own line", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Structs: []*emit.Struct{{
				Name: "Combined", Package: "x", Target: target,
				Embeds: []*emit.Embed{
					{Type: emit.External("io", "Reader")},
					{Type: emit.External("io", "Closer")},
				},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "io.Reader\n\tio.Closer\n") {
			t.Fatalf("embeds should render adjacent on separate lines; got:\n%s", body)
		}
	})
}

// TestTypeParams_Render covers generic-decl rendering via the
// renderTypeParams helper across struct / interface / alias.
func TestTypeParams_Render(t *testing.T) {
	t.Parallel()

	t.Run("generic struct with 'any' constraint renders [T any]", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Structs: []*emit.Struct{{
				Name: "Box", Package: "x", Target: target,
				TypeParams: []*emit.TypeParam{{Name: "T"}},
				Fields:     []*emit.Field{{Name: "Value", Type: emit.Builtin("int")}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type Box[T any] struct {") {
			t.Fatalf("generic struct mismatched; got:\n%s", body)
		}
	})

	t.Run("generic struct with builtin constraint", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Structs: []*emit.Struct{{
				Name: "Box", Package: "x", Target: target,
				TypeParams: []*emit.TypeParam{{
					Name: "T",
					Constraint: &emit.Constraint{
						Embedded: []emit.Ref{emit.Builtin("comparable")},
					},
				}},
				Fields: []*emit.Field{{Name: "Value", Type: emit.Builtin("int")}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type Box[T comparable] struct {") {
			t.Fatalf("constrained generic mismatched; got:\n%s", body)
		}
	})

	t.Run("generic interface renders bracketed params after the name", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Interfaces: []*emit.Interface{{
				Name: "Container", Package: "x", Target: target,
				TypeParams: []*emit.TypeParam{{Name: "T"}, {Name: "U"}},
				Methods:    []*emit.Method{{Name: "Add"}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type Container[T any, U any] interface {") {
			t.Fatalf("multi-param generic interface mismatched; got:\n%s", body)
		}
	})

	t.Run("generic alias renders params after the name", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Aliases: []*emit.Alias{{
				Name: "List", Package: "x", File: target,
				TypeParams: []*emit.TypeParam{{Name: "T"}},
				Target:     emit.Builtin("int"),
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type List[T any] int") {
			t.Fatalf("generic alias mismatched; got:\n%s", body)
		}
	})
}

// TestRenderType_Union covers the ShapeUnion path: type-set
// constraints (`A | B | ~C`) render via the union shape with `~`
// prefixing approximation terms.
func TestRenderType_Union(t *testing.T) {
	t.Parallel()

	t.Run("generic struct with union constraint", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "x", Path: "x",
			Structs: []*emit.Struct{{
				Name: "Numeric", Package: "x", Target: target,
				TypeParams: []*emit.TypeParam{{
					Name: "T",
					Constraint: &emit.Constraint{
						Embedded: []emit.Ref{
							emit.Union(
								emit.UnionTerm{Type: emit.Builtin("int"), Approx: true},
								emit.UnionTerm{Type: emit.Builtin("float64"), Approx: true},
								emit.UnionTerm{Type: emit.Builtin("string")},
							),
						},
					},
				}},
				Fields: []*emit.Field{{Name: "V", Type: emit.Builtin("int")}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		if !strings.Contains(string(body), "type Numeric[T ~int | ~float64 | string]") {
			t.Fatalf("union constraint mismatched; got:\n%s", body)
		}
	})
}

// TestFieldTagAggregation covers the renderFields enhancement: the
// `Field.Tag` base string plus `*emit.Tag` entries appended to
// `Field.Tags()` aggregate into a single backtick blob.
func TestFieldTagAggregation(t *testing.T) {
	t.Parallel()

	t.Run("base Tag alone renders inside backticks", func(t *testing.T) {
		t.Parallel()
		body := renderTagsForStruct(t, &emit.Field{
			Name: "ID",
			Type: emit.Builtin("int"),
			Tag:  `json:"id"`,
		})
		if !strings.Contains(body, "`json:\"id\"`") {
			t.Fatalf("base tag should appear in backticks; got:\n%s", body)
		}
	})

	t.Run("slot Tag entries append after base Tag in order", func(t *testing.T) {
		t.Parallel()
		f := &emit.Field{
			Name: "ID",
			Type: emit.Builtin("int"),
			Tag:  `json:"id"`,
		}
		if err := f.Tags().Append(&emit.Tag{Key: "db", Value: "id"}, emit.Provenance{}); err != nil {
			t.Fatalf("Append db tag: %v", err)
		}
		if err := f.Tags().Append(&emit.Tag{Key: "yaml", Value: "id"}, emit.Provenance{}); err != nil {
			t.Fatalf("Append yaml tag: %v", err)
		}
		body := renderTagsForStruct(t, f)
		if !strings.Contains(body, "`json:\"id\" db:\"id\" yaml:\"id\"`") {
			t.Fatalf("aggregated tag blob mismatched; got:\n%s", body)
		}
	})

	t.Run("slot Tag values are Go-string-escaped", func(t *testing.T) {
		t.Parallel()
		f := &emit.Field{Name: "X", Type: emit.Builtin("string")}
		if err := f.Tags().Append(&emit.Tag{Key: "msg", Value: `with "quotes"`}, emit.Provenance{}); err != nil {
			t.Fatalf("Append msg tag: %v", err)
		}
		body := renderTagsForStruct(t, f)
		if !strings.Contains(body, `msg:"with \"quotes\""`) {
			t.Fatalf("tag value should be Go-string-escaped; got:\n%s", body)
		}
	})

	t.Run("field with neither base nor slot tag renders no backticks", func(t *testing.T) {
		t.Parallel()
		body := renderTagsForStruct(t, &emit.Field{Name: "Plain", Type: emit.Builtin("int")})
		if strings.Contains(body, "`") {
			t.Fatalf("empty tag should not produce backticks; got:\n%s", body)
		}
	})
}

// renderTagsForStruct builds a single-field struct with the
// supplied field, renders it, and returns the rendered file body.
// Used by TestFieldTagAggregation to assert the tag blob shape
// through the public render path.
func renderTagsForStruct(t *testing.T, f *emit.Field) string {
	t.Helper()
	ctx, mem, d := newBackendContext(t)
	target := emit.Target{Dir: "x", Filename: "x.go", Package: "x"}
	addEmitPackage(t, ctx, &emit.Package{
		Name: "x", Path: "x",
		Structs: []*emit.Struct{{
			Name: "S", Package: "x", Target: target,
			Fields: []*emit.Field{f},
		}},
	})
	body := assertRenderSucceeds(t, ctx, mem, d, target)
	return string(body)
}

// TestSubElements_Golden pins canonical output for each Phase F
// shape: struct embeds, generic decls, union constraints, and
// aggregated field tags.
func TestSubElements_Golden(t *testing.T) {
	t.Parallel()

	t.Run("struct_embeds — adjacent embeds + fields", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "iox", Filename: "wrapper.go", Package: "iox"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "iox", Path: "iox",
			Structs: []*emit.Struct{{
				BaseEmit: emit.BaseEmit{DocLines: []string{"Wrapper composes Reader and Closer."}},
				Name:     "Wrapper", Package: "iox", Target: target,
				Embeds: []*emit.Embed{
					{Type: emit.External("io", "Reader")},
					{Type: emit.External("io", "Closer")},
				},
				Fields: []*emit.Field{
					{Name: "Closed", Type: emit.Builtin("bool")},
				},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "struct_embeds.go.golden"))
	})

	t.Run("generic_struct — single-term constraint", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "containers", Filename: "box.go", Package: "containers"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "containers", Path: "containers",
			Structs: []*emit.Struct{{
				BaseEmit: emit.BaseEmit{DocLines: []string{"Box holds a single comparable value."}},
				Name:     "Box", Package: "containers", Target: target,
				TypeParams: []*emit.TypeParam{{
					Name:       "T",
					Constraint: &emit.Constraint{Embedded: []emit.Ref{emit.Builtin("comparable")}},
				}},
				Fields: []*emit.Field{{Name: "V", Type: emit.Builtin("int")}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "generic_struct.go.golden"))
	})

	t.Run("generic_union — type-set constraint with approx terms", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "math", Filename: "ord.go", Package: "math"}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "math", Path: "math",
			Structs: []*emit.Struct{{
				Name: "Ordered", Package: "math", Target: target,
				TypeParams: []*emit.TypeParam{{
					Name: "T",
					Constraint: &emit.Constraint{
						Embedded: []emit.Ref{
							emit.Union(
								emit.UnionTerm{Type: emit.Builtin("int"), Approx: true},
								emit.UnionTerm{Type: emit.Builtin("float64"), Approx: true},
								emit.UnionTerm{Type: emit.Builtin("string")},
							),
						},
					},
				}},
				Fields: []*emit.Field{{Name: "V", Type: emit.Builtin("int")}},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "generic_union.go.golden"))
	})

	t.Run("field_tag_aggregation — base + slot contributors", func(t *testing.T) {
		t.Parallel()
		ctx, mem, d := newBackendContext(t)
		target := emit.Target{Dir: "users", Filename: "user.go", Package: "users"}
		idField := &emit.Field{Name: "ID", Type: emit.Builtin("int"), Tag: `json:"id"`}
		if err := idField.Tags().Append(&emit.Tag{Key: "db", Value: "user_id"}, emit.Provenance{}); err != nil {
			t.Fatalf("Append db tag: %v", err)
		}
		if err := idField.Tags().Append(&emit.Tag{Key: "yaml", Value: "id"}, emit.Provenance{}); err != nil {
			t.Fatalf("Append yaml tag: %v", err)
		}
		nameField := &emit.Field{Name: "Name", Type: emit.Builtin("string")}
		validateTag := &emit.Tag{Key: "validate", Value: "required,max=64"}
		if err := nameField.Tags().Append(validateTag, emit.Provenance{}); err != nil {
			t.Fatalf("Append validate tag: %v", err)
		}
		addEmitPackage(t, ctx, &emit.Package{
			Name: "users", Path: "users",
			Structs: []*emit.Struct{{
				BaseEmit: emit.BaseEmit{DocLines: []string{"User carries the canonical user record."}},
				Name:     "User", Package: "users", Target: target,
				Fields: []*emit.Field{idField, nameField},
			}},
		})
		body := assertRenderSucceeds(t, ctx, mem, d, target)
		testpipe.MatchesGoldenBytes(t, body, goldenPath(t, "field_tag_aggregation.go.golden"))
	})
}
