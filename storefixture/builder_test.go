// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package storefixture_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/eidos/storefixture"
	"go.thesmos.sh/eidos/store"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("seeds a default-named, default-pathed empty package", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New()
		p := b.PackageNode()
		if p == nil {
			t.Fatalf("PackageNode returned nil")
		}
		if p.Name != "test" {
			t.Fatalf("default name should be %q; got %q", "test", p.Name)
		}
		if p.Path != "example.com/test" {
			t.Fatalf("default path should be %q; got %q", "example.com/test", p.Path)
		}
		if len(p.Structs) != 0 || len(p.Interfaces) != 0 || len(p.Functions) != 0 {
			t.Fatalf("fresh builder should hold no declarations")
		}
	})
}

func TestBuilder_Package(t *testing.T) {
	t.Parallel()

	t.Run("overwrites name and path on an empty builder", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New().Package("users", "example.com/users")
		p := b.PackageNode()
		if p.Name != "users" || p.Path != "example.com/users" {
			t.Fatalf("Package did not overwrite identity: %+v", p)
		}
	})

	t.Run("rewrites the qualified-name path on every existing decl", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New().
			Struct("S", nil).
			Interface("I", nil).
			Function("F", nil).
			Variable("V", nil).
			Constant("C", nil).
			Enum("E", nil).
			Alias("A", nil).
			Package("users", "example.com/users")

		p := b.PackageNode()
		requireQName(t, p.Structs[0].QName(), "example.com/users.S")
		requireQName(t, p.Interfaces[0].QName(), "example.com/users.I")
		requireQName(t, p.Functions[0].QName(), "example.com/users.F")
		requireQName(t, p.Variables[0].QName(), "example.com/users.V")
		requireQName(t, p.Constants[0].QName(), "example.com/users.C")
		requireQName(t, p.Enums[0].QName(), "example.com/users.E")
		requireQName(t, p.Aliases[0].QName(), "example.com/users.A")
	})

	t.Run("returns the same builder for chaining", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New()
		if got := b.Package("a", "a"); got != b {
			t.Fatalf("Package should return its receiver for chaining")
		}
	})
}

func TestBuilder_Import(t *testing.T) {
	t.Parallel()

	t.Run("records the import on the package", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New().Import("context").Import("time")
		imps := b.PackageNode().Imports
		if len(imps) != 2 {
			t.Fatalf("expected 2 imports; got %d", len(imps))
		}
		if imps[0].Path != "context" || imps[1].Path != "time" {
			t.Fatalf("imports wrong: %+v", imps)
		}
		if imps[0].Owner == nil {
			t.Fatalf("import owner back-pointer should be wired")
		}
	})
}

func TestBuilder_Build(t *testing.T) {
	t.Parallel()

	t.Run("returns a populated store with the configured package", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().Struct("User", nil).Build()
		got, ok := s.Nodes().Structs().ByQName("example.com/test.User")
		if !ok {
			t.Fatalf("struct should be indexed by qname")
		}
		if got.Name != "User" {
			t.Fatalf("struct name wrong: %q", got.Name)
		}
	})

	t.Run("returns independent stores across calls", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New().Struct("X", nil)
		s1, s2 := b.Build(), b.Build()
		if s1 == s2 {
			t.Fatalf("Build must return fresh stores per call")
		}
	})

	t.Run("panics on duplicate qualified names", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New().Struct("Dup", nil).Struct("Dup", nil)
		rec := requirePanic(t, func() { b.Build() })
		err, ok := rec.(error)
		if !ok {
			t.Fatalf("recovered value should be error; got %T", rec)
		}
		if !errors.Is(err, store.ErrDuplicateQName) {
			t.Fatalf("expected ErrDuplicateQName; got %v", err)
		}
	})
}

func TestBuilder_PackageNode(t *testing.T) {
	t.Parallel()

	t.Run("returns the same aliased package across calls", func(t *testing.T) {
		t.Parallel()
		b := storefixture.New()
		first := b.PackageNode()
		second := b.PackageNode()
		if first != second {
			t.Fatalf("PackageNode must alias the builder's internal pointer")
		}
	})
}
