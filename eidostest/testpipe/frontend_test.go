// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package testpipe_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/eidostest/testpipe"
	"go.thesmos.sh/eidos/store"
)

func TestFromNodes(t *testing.T) {
	t.Parallel()

	t.Run("returns a frontend whose Name identifies the synthetic source", func(t *testing.T) {
		t.Parallel()
		fe := testpipe.FromNodes()
		if fe.Name() == "" {
			t.Fatalf("FromNodes frontend must have a non-empty Name")
		}
	})

	t.Run("Load adds every supplied package on the first call", func(t *testing.T) {
		t.Parallel()
		pkg := storefixture.New().
			Package("users", "example.com/users").
			Struct("User", nil).
			PackageNode()

		fe := testpipe.FromNodes(pkg)
		s := store.New()
		if err := fe.Load("", s, diag.Discard()); err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if _, ok := s.Nodes().Structs().ByQName("example.com/users.User"); !ok {
			t.Fatalf("package not added to store")
		}
	})

	t.Run("Load is idempotent across repeated invocations", func(t *testing.T) {
		t.Parallel()
		pkg := storefixture.New().Struct("S", nil).PackageNode()
		fe := testpipe.FromNodes(pkg)
		s := store.New()

		if err := fe.Load("first", s, diag.Discard()); err != nil {
			t.Fatalf("first Load returned error: %v", err)
		}
		if err := fe.Load("second", s, diag.Discard()); err != nil {
			t.Fatalf("second Load returned error: %v", err)
		}
		if got := s.Nodes().Structs().Len(); got != 1 {
			t.Fatalf("Load should add exactly once across calls; got %d structs", got)
		}
	})

	t.Run("Load propagates AddPackage errors", func(t *testing.T) {
		t.Parallel()
		pkg := storefixture.New().Struct("Dup", nil).PackageNode()
		fe := testpipe.FromNodes(pkg, pkg)
		s := store.New()

		if err := fe.Load("", s, diag.Discard()); err == nil {
			t.Fatalf("expected error from duplicate-qname Load; got nil")
		}
	})

	t.Run("accepts multiple packages", func(t *testing.T) {
		t.Parallel()
		a := storefixture.New().Package("a", "example.com/a").Struct("A", nil).PackageNode()
		b := storefixture.New().Package("b", "example.com/b").Struct("B", nil).PackageNode()

		fe := testpipe.FromNodes(a, b)
		s := store.New()
		if err := fe.Load("", s, diag.Discard()); err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if _, ok := s.Nodes().Structs().ByQName("example.com/a.A"); !ok {
			t.Fatalf("package a not loaded")
		}
		if _, ok := s.Nodes().Structs().ByQName("example.com/b.B"); !ok {
			t.Fatalf("package b not loaded")
		}
	})
}
