// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"testing"

	"go.thesmos.sh/eidos/store"
)

func TestNewReader(t *testing.T) {
	t.Parallel()

	t.Run("returns a Reader with empty ReadSet", func(t *testing.T) {
		t.Parallel()
		r := store.NewReader(store.New())
		if r.ReadSet() == nil {
			t.Fatalf("ReadSet should be non-nil")
		}
		if r.ReadSet().Len() != 0 {
			t.Fatalf("new Reader's ReadSet should be empty")
		}
	})
}

func TestReader_Store(t *testing.T) {
	t.Parallel()

	t.Run("returns the underlying Store", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		r := store.NewReader(s)
		if r.Store() != s {
			t.Fatalf("Store() should return the wrapped instance")
		}
	})
}

func TestReader_ReadSet(t *testing.T) {
	t.Parallel()

	t.Run("returns the same instance across calls", func(t *testing.T) {
		t.Parallel()
		r := store.NewReader(store.New())
		if a, b := r.ReadSet(), r.ReadSet(); a != b {
			t.Fatalf("ReadSet should be the same instance across calls")
		}
	})
}

func TestReader_NodeQueries(t *testing.T) {
	t.Parallel()

	t.Run("each node-side accessor returns a query that records its own tag", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		assertNoError(t, s.Nodes().AddPackage(makeUserPackage()))
		r := store.NewReader(s)
		_ = r.Packages().Slice()
		_ = r.Files().Slice()
		_ = r.Imports().Slice()
		_ = r.Structs().Slice()
		_ = r.Interfaces().Slice()
		_ = r.Methods().Slice()
		_ = r.Fields().Slice()
		_ = r.Functions().Slice()
		_ = r.Variables().Slice()
		_ = r.Constants().Slice()
		_ = r.Enums().Slice()
		_ = r.EnumVariants().Slice()
		_ = r.Aliases().Slice()
		want := []string{
			"node:packages", "node:files", "node:imports", "node:structs",
			"node:interfaces", "node:methods", "node:fields", "node:functions",
			"node:variables", "node:constants", "node:enums", "node:enum_variants",
			"node:aliases",
		}
		for _, k := range want {
			if !r.ReadSet().Has(k) {
				t.Fatalf("ReadSet missing key %q", k)
			}
		}
	})
}

func TestReader_EmitQueries(t *testing.T) {
	t.Parallel()

	t.Run("each emit-side accessor returns a query that records its own tag", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		assertNoError(t, s.Emit().AddPackage(makeUserEmitPackage()))
		r := store.NewReader(s)
		_ = r.EmitPackages().Slice()
		_ = r.EmitFiles().Slice()
		_ = r.EmitImports().Slice()
		_ = r.EmitStructs().Slice()
		_ = r.EmitInterfaces().Slice()
		_ = r.EmitMethods().Slice()
		_ = r.EmitFields().Slice()
		_ = r.EmitFunctions().Slice()
		_ = r.EmitVariables().Slice()
		_ = r.EmitConstants().Slice()
		_ = r.EmitEnums().Slice()
		_ = r.EmitEnumVariants().Slice()
		_ = r.EmitAliases().Slice()
		want := []string{
			"emit:packages", "emit:files", "emit:imports", "emit:structs",
			"emit:interfaces", "emit:methods", "emit:fields", "emit:functions",
			"emit:variables", "emit:constants", "emit:enums", "emit:enum_variants",
			"emit:aliases",
		}
		for _, k := range want {
			if !r.ReadSet().Has(k) {
				t.Fatalf("ReadSet missing key %q", k)
			}
		}
	})
}
