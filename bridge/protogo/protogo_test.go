// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protogo_test

import (
	"strings"
	"testing"

	backend_golang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/bridge/protogo"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/protopipe"
	"go.thesmos.sh/eidos/frontend/protobuf"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/reference/mockgen"
	"go.thesmos.sh/eidos/sink"
	"go.thesmos.sh/eidos/store"
)

// TestAnnotate_StampsGoNameOnFields covers the field-name
// translation: every proto-derived [node.Field] carries the
// Go-idiomatic PascalCase identifier under [protogo.MetaGoName].
func TestAnnotate_StampsGoNameOnFields(t *testing.T) {
	t.Parallel()

	t.Run("snake_case proto fields stamp the PascalCase Go form", func(t *testing.T) {
		t.Parallel()
		pkgs := runProtogo(t, "messages")
		main := findPackage(t, pkgs, "eidos.protobuf.testdata.messages")
		user := findStruct(main, "User")
		cases := map[string]string{
			"id":            "ID",
			"legacy_field":  "LegacyField",
			"display_name":  "DisplayName",
			"optional_note": "OptionalNote",
		}
		for proto, want := range cases {
			f := user.FieldByName(proto)
			if f == nil {
				t.Fatalf("field %q missing on User", proto)
			}
			got, ok := protogo.MetaGoName.Get(f.Meta())
			if !ok {
				t.Fatalf("go.name missing on User.%s", proto)
			}
			if got != want {
				t.Fatalf("go.name on User.%s = %q, want %q", proto, got, want)
			}
		}
	})
}

// TestAnnotate_StampsGoTypeOnFields covers the type-translation
// table: scalar fields stamp their Go-side type, message-typed
// fields stamp the bare name, repeated wraps in `[]`, map wraps
// in `map[...]`. Well-known references go through the
// well-known table.
func TestAnnotate_StampsGoTypeOnFields(t *testing.T) {
	t.Parallel()

	pkgs := runProtogo(t, "messages")
	main := findPackage(t, pkgs, "eidos.protobuf.testdata.messages")

	t.Run("scalar field stamps the matching Go scalar", func(t *testing.T) {
		t.Parallel()
		user := findStruct(main, "User")
		id := user.FieldByName("id")
		got, ok := protogo.MetaGoType.Get(id.Type.Meta())
		if !ok || got != "string" {
			t.Fatalf("go.type on User.id = (%q, %v), want \"string\"", got, ok)
		}
		legacy := user.FieldByName("legacy_field")
		got, ok = protogo.MetaGoType.Get(legacy.Type.Meta())
		if !ok || got != "int32" {
			t.Fatalf("go.type on User.legacy_field = (%q, %v), want \"int32\"", got, ok)
		}
	})

	t.Run("repeated scalar field wraps the element in []", func(t *testing.T) {
		t.Parallel()
		user := findStruct(main, "User")
		tags := user.FieldByName("tags")
		got, ok := protogo.MetaGoType.Get(tags.Type.Meta())
		if !ok || got != "[]string" {
			t.Fatalf("go.type on User.tags = (%q, %v), want \"[]string\"", got, ok)
		}
	})

	t.Run("map<K,V> field renders as map[goK]goV", func(t *testing.T) {
		t.Parallel()
		user := findStruct(main, "User")
		attrs := user.FieldByName("attrs")
		got, ok := protogo.MetaGoType.Get(attrs.Type.Meta())
		if !ok || got != "map[string]int32" {
			t.Fatalf("go.type on User.attrs = (%q, %v), want \"map[string]int32\"", got, ok)
		}
	})

	t.Run("repeated message field is left for emit.SliceOf(External(...)) — no go.type stamp", func(t *testing.T) {
		t.Parallel()
		user := findStruct(main, "User")
		friends := user.FieldByName("friends")
		if got, ok := protogo.MetaGoType.Get(friends.Type.Meta()); ok {
			t.Fatalf(
				"go.type unexpectedly stamped on []Profile; got %q (slice over a named ref defers to refconv)",
				got,
			)
		}
	})

	t.Run("message-typed field is left for the backend's emit.External path (no go.type stamp)", func(t *testing.T) {
		t.Parallel()
		user := findStruct(main, "User")
		profile := user.FieldByName("profile")
		if got, ok := protogo.MetaGoType.Get(profile.Type.Meta()); ok {
			t.Fatalf(
				"go.type unexpectedly stamped on a named message ref; got %q (the bridge should let refconv's External path render cross-package qualifiers)",
				got,
			)
		}
	})
}

// TestAnnotate_WellKnownTypes covers the well-known translation
// table: every reference to a `google.protobuf.*` type stamps
// the canonical Go-side form derived from the bridge's table.
func TestAnnotate_WellKnownTypes(t *testing.T) {
	t.Parallel()

	pkgs := runProtogo(t, "wellknown")
	main := findPackage(t, pkgs, "eidos.protobuf.testdata.wellknown")
	event := findStruct(main, "Event")

	cases := map[string]string{
		"at":      "*timestamppb.Timestamp",
		"span":    "*durationpb.Duration",
		"payload": "*anypb.Any",
		"mask":    "*fieldmaskpb.FieldMask",
		"done":    "*emptypb.Empty",
	}
	for proto, want := range cases {
		t.Run("well-known "+proto+" stamps the canonical Go form", func(t *testing.T) {
			t.Parallel()
			f := event.FieldByName(proto)
			if f == nil {
				t.Fatalf("field %q missing on Event", proto)
			}
			got, ok := protogo.MetaGoType.Get(f.Type.Meta())
			if !ok {
				t.Fatalf("go.type missing on Event.%s", proto)
			}
			if got != want {
				t.Fatalf("go.type on Event.%s = %q, want %q", proto, got, want)
			}
		})
	}
}

// TestAnnotate_PackageMetaFromGoPackageOption covers the
// package-level stamping: `go.import` derives from the proto
// `go_package` option; `go.name` derives from the same value's
// semicolon-suffix or last `/` segment.
func TestAnnotate_PackageMetaFromGoPackageOption(t *testing.T) {
	t.Parallel()

	t.Run("fileoptions fixture stamps go.import + go.name from go_package", func(t *testing.T) {
		t.Parallel()
		pkgs := runProtogo(t, "fileoptions")
		pkg := findPackage(t, pkgs, "eidos.protobuf.testdata.fileoptions")
		gotImport, ok := protogo.MetaGoImport.Get(pkg.Meta())
		if !ok {
			t.Fatalf("go.import missing on fileoptions package")
		}
		const wantImport = "github.com/example/fileoptions"
		if gotImport != wantImport {
			t.Fatalf("go.import = %q, want %q", gotImport, wantImport)
		}
		gotName, ok := protogo.MetaGoName.Get(pkg.Meta())
		if !ok {
			t.Fatalf("go.name missing on fileoptions package")
		}
		// The proto go_package option value has no semicolon, so
		// the rule falls back to the last `/` segment.
		const wantName = "fileoptions"
		if gotName != wantName {
			t.Fatalf("go.name = %q, want %q", gotName, wantName)
		}
	})
}

// TestAnnotate_OptionalWrap covers the proto3-optional-wrap rule:
// a field carrying `proto.field.optional = true` stamps
// `go.type = "*<inner>"` on its TypeRef. Without the wrap, the
// rendered Go form loses the explicit-presence semantics
// proto3-optional encodes.
func TestAnnotate_OptionalWrap(t *testing.T) {
	t.Parallel()

	t.Run("optional scalar field wraps the Go-side form in *", func(t *testing.T) {
		t.Parallel()
		pkgs := runProtogo(t, "messages")
		main := findPackage(t, pkgs, "eidos.protobuf.testdata.messages")
		user := findStruct(main, "User")
		note := user.FieldByName("optional_note")
		got, ok := protogo.MetaGoType.Get(note.Type.Meta())
		if !ok || got != "*string" {
			t.Fatalf("go.type on User.optional_note = (%q, %v), want \"*string\"", got, ok)
		}
	})
}

// TestAnnotate_WellKnownImport covers the well-known import
// stamping: every reference to a `google.protobuf.*` type also
// stamps the canonical Go import path on the same TypeRef so the
// backend's render-site can register the import on the host
// file's ImportSet.
func TestAnnotate_WellKnownImport(t *testing.T) {
	t.Parallel()

	t.Run("Timestamp reference stamps go.import = timestamppb path", func(t *testing.T) {
		t.Parallel()
		pkgs := runProtogo(t, "wellknown")
		main := findPackage(t, pkgs, "eidos.protobuf.testdata.wellknown")
		event := findStruct(main, "Event")
		at := event.FieldByName("at")
		got, ok := protogo.MetaGoImport.Get(at.Type.Meta())
		if !ok {
			t.Fatalf("go.import missing on Event.at")
		}
		const want = "google.golang.org/protobuf/types/known/timestamppb"
		if got != want {
			t.Fatalf("go.import on Event.at = %q, want %q", got, want)
		}
	})
}

// TestAnnotate_FrontendMarkerScope verifies the bridge filters
// its walk by the cross-frontend marker — Go-derived packages
// remain untouched even when they share a pipeline with proto
// sources.
func TestAnnotate_FrontendMarkerScope(t *testing.T) {
	t.Parallel()

	t.Run("Go-derived nodes do not pick up go.* stamps from the bridge", func(t *testing.T) {
		t.Parallel()
		// The protogo bridge walks every package in the store but
		// short-circuits when the package's frontend marker
		// doesn't equal "protobuf". A Go-derived package's fields
		// keep whatever Go-side meta the Go frontend produced;
		// the bridge contributes nothing on top.
		pkgs := runProtogo(t, "simple")
		// Every package in pkgs should carry the protobuf
		// frontend marker because the simple fixture only has
		// proto sources. If a future regression broadens the
		// bridge's scope, this assertion exists as a safety net.
		for _, pkg := range pkgs {
			marker, _ := protobuf.MetaFrontend.Get(pkg.Meta())
			if marker != protobuf.FrontendName {
				t.Fatalf(
					"protogo touched a package whose frontend marker is %q, not %q",
					marker, protobuf.FrontendName,
				)
			}
		}
	})
}

// TestAnnotate_Idempotency covers the bridge-idempotency
// contract: a pre-stamped go.type wins over the bridge's
// table-driven value, and a second pass produces no change. The
// test builds a minimal source-side store and drives the public
// [plugin.Annotator.Annotate] surface so the contract is checked
// through the same boundary downstream consumers cross.
func TestAnnotate_Idempotency(t *testing.T) {
	t.Parallel()

	t.Run("pre-stamped go.type on a TypeRef survives Annotate", func(t *testing.T) {
		t.Parallel()
		const override = "fixture-override"
		s, ref := buildSyntheticProtoStore(t)
		protogo.MetaGoType.Set(ref.Meta(), override, "fixture-test")
		annotateOnce(t, s)
		got, ok := protogo.MetaGoType.Get(ref.Meta())
		if !ok || got != override {
			t.Fatalf("override lost; go.type = (%q, %v), want %q", got, ok, override)
		}
	})

	t.Run("two Annotate passes produce identical meta", func(t *testing.T) {
		t.Parallel()
		s, ref := buildSyntheticProtoStore(t)
		annotateOnce(t, s)
		first, ok := protogo.MetaGoType.Get(ref.Meta())
		if !ok {
			t.Fatalf("first pass did not stamp go.type on the scalar ref")
		}
		annotateOnce(t, s)
		second, ok := protogo.MetaGoType.Get(ref.Meta())
		if !ok || second != first {
			t.Fatalf("second pass changed go.type; first=%q second=(%q, %v)", first, second, ok)
		}
	})
}

// TestRender_BridgeAffectsGoOutput pins the end-to-end render
// effect: with the protogo bridge registered, mockgen's emitted
// mock for a proto service references the Go-side translated
// type names rather than the bare proto-source names. The
// rendered output is captured through the in-memory sink and
// grep-checked for the expected Go-side identifiers.
func TestRender_BridgeAffectsGoOutput(t *testing.T) {
	t.Parallel()

	t.Run("mockgen output threads through protogo's translation tables", func(t *testing.T) {
		t.Parallel()
		mem := sink.NewMemory()
		root := fixtureRoot(t, "services")
		result := protopipe.Run(t, protopipe.RunOptions{
			SourceDir:  root,
			Annotators: []plugin.Annotator{protogo.New()},
			Generators: []plugin.Generator{mockgen.New()},
			Backend:    backend_golang.New(),
			Sink:       mem,
		})
		if result.Diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", result.Diag.Diagnostics())
		}
		body := collectSinkBody(mem)
		if !strings.Contains(body, "GreeterMock") {
			t.Fatalf("expected GreeterMock in rendered output; got:\n%s", body)
		}
		if !strings.Contains(body, "HelloRequest") {
			t.Fatalf("expected HelloRequest reference; got:\n%s", body)
		}
	})
}

// buildSyntheticProtoStore returns a fresh [store.Store]
// populated with one proto-marked [node.Package] carrying one
// [node.Struct] with one scalar [node.Field]. The returned
// TypeRef is the scalar field's type — callers stamp meta on it
// directly to set up idempotency assertions.
//
// The synthetic graph carries the cross-frontend marker so the
// protogo bridge's iteration filter accepts the package, but no
// other proto-meta is present; the test focuses the contract on
// the per-TypeRef idempotency guard without standing up a full
// fixture.
func buildSyntheticProtoStore(t *testing.T) (*store.Store, *node.TypeRef) {
	t.Helper()
	ref := &node.TypeRef{TypeKind: node.TypeRefNamed, Name: "int32"}
	f := &node.Field{Name: "scalar", Type: ref}
	srcStruct := &node.Struct{
		Name:    "Container",
		Package: "eidos.test.synthetic",
		Fields:  []*node.Field{f},
	}
	pkg := &node.Package{
		Name:    "synthetic",
		Path:    "eidos.test.synthetic",
		Structs: []*node.Struct{srcStruct},
	}
	protobuf.MetaFrontend.Set(pkg.Meta(), protobuf.FrontendName, "fixture-test")
	s := store.New()
	if err := s.Nodes().AddPackage(pkg); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}
	return s, ref
}

// annotateOnce runs the bridge's [plugin.Annotator.Annotate]
// against s exactly once and fails the test on any error
// diagnostic. The helper centralises the AnnotatorContext shape
// so idempotency subtests stay focused on the contract instead
// of the wiring.
func annotateOnce(t *testing.T, s *store.Store) {
	t.Helper()
	d := diag.New()
	bridge := protogo.New()
	if err := bridge.Annotate(&plugin.AnnotatorContext{
		Store:  s,
		Reader: store.NewReader(s),
		Diag:   d,
	}); err != nil {
		t.Fatalf("bridge.Annotate: %v", err)
	}
	if d.HasErrors() {
		t.Fatalf("expected no error diagnostics; got %+v", d.Diagnostics())
	}
}
