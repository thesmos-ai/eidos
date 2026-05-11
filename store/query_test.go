// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/store"
)

func makeStructPopulatedReader(t *testing.T) *store.Reader {
	t.Helper()
	s := store.New()
	assertNoError(t, s.Nodes().AddPackage(makeUserPackage()))
	return store.NewReader(s)
}

func TestQuery_Where(t *testing.T) {
	t.Parallel()

	t.Run("filters by the supplied predicate", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		got := r.Structs().
			Where(func(s *node.Struct) bool { return s.Name == "User" }).
			Slice()
		if len(got) != 1 || got[0].Name != "User" {
			t.Fatalf("Where filter mismatch: %+v", got)
		}
	})

	t.Run("multiple Where calls compose as logical AND", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		got := r.Structs().
			Where(func(s *node.Struct) bool { return s.Package != "" }).
			Where(func(s *node.Struct) bool { return s.Name == "User" }).
			Slice()
		if len(got) != 1 || got[0].Name != "User" {
			t.Fatalf("composed Where mismatch: %+v", got)
		}
	})

	t.Run("nil predicate is a no-op", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		got := r.Structs().Where(nil).Slice()
		if len(got) != 2 {
			t.Fatalf("nil Where should not filter; got %d", len(got))
		}
	})

	t.Run("returns a new Query, leaving the original usable", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		original := r.Structs()
		filtered := original.Where(func(s *node.Struct) bool { return s.Name == "User" })
		if filtered == original {
			t.Fatalf("Where should return a new Query")
		}
		if got := original.Slice(); len(got) != 2 {
			t.Fatalf("original Query should remain unfiltered; got %d", len(got))
		}
	})
}

func TestQuery_Each(t *testing.T) {
	t.Parallel()

	t.Run("invokes fn for every match in insertion order", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		var seen []string
		r.Structs().Each(func(s *node.Struct) { seen = append(seen, s.Name) })
		if !slices.Equal(seen, []string{"User", "Address"}) {
			t.Fatalf("Each order = %v, want [User Address]", seen)
		}
	})

	t.Run("respects the accumulated predicate", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		var seen []string
		r.Structs().
			Where(func(s *node.Struct) bool { return s.Name == "Address" }).
			Each(func(s *node.Struct) { seen = append(seen, s.Name) })
		if !slices.Equal(seen, []string{"Address"}) {
			t.Fatalf("filtered Each mismatch: %v", seen)
		}
	})

	t.Run("records the source tag in the ReadSet", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		r.Structs().Each(func(*node.Struct) {})
		if !r.ReadSet().Has("node:structs") {
			t.Fatalf("Each should record the source tag")
		}
	})
}

func TestQuery_Slice(t *testing.T) {
	t.Parallel()

	t.Run("returns matched items in source order", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		got := r.Structs().Slice()
		if len(got) != 2 || got[0].Name != "User" || got[1].Name != "Address" {
			t.Fatalf("Slice order mismatch: %+v", got)
		}
	})

	t.Run("returns an empty slice when nothing matches", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		got := r.Structs().Where(func(*node.Struct) bool { return false }).Slice()
		if len(got) != 0 {
			t.Fatalf("expected empty slice; got %d items", len(got))
		}
	})
}

func TestQuery_First(t *testing.T) {
	t.Parallel()

	t.Run("returns the first match and true", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		got, ok := r.Structs().First()
		if !ok || got.Name != "User" {
			t.Fatalf("First mismatch: %+v ok=%v", got, ok)
		}
	})

	t.Run("returns zero value and false when nothing matches", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		got, ok := r.Structs().Where(func(*node.Struct) bool { return false }).First()
		if ok || got != nil {
			t.Fatalf("First with no match should return (nil, false); got %+v ok=%v", got, ok)
		}
	})
}

func TestQuery_Count(t *testing.T) {
	t.Parallel()

	t.Run("returns the total number of source items when no predicate", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		if got := r.Structs().Count(); got != 2 {
			t.Fatalf("Count = %d, want 2", got)
		}
	})

	t.Run("returns the count of matched items under a predicate", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		got := r.Structs().
			Where(func(s *node.Struct) bool { return s.Name == "User" }).
			Count()
		if got != 1 {
			t.Fatalf("Count(filter=User) = %d, want 1", got)
		}
	})
}

func TestQuery_RecordsReadOnTerminalsOnly(t *testing.T) {
	t.Parallel()

	t.Run("Where alone does not record", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		_ = r.Structs().Where(func(*node.Struct) bool { return true })
		if r.ReadSet().Len() != 0 {
			t.Fatalf("Where should not record reads; got Len=%d", r.ReadSet().Len())
		}
	})

	t.Run("each terminal records exactly once per call", func(t *testing.T) {
		t.Parallel()
		r := makeStructPopulatedReader(t)
		_ = r.Structs().Slice()
		_ = r.Structs().Slice()
		// Same tag, idempotent in the read-set.
		if r.ReadSet().Len() != 1 {
			t.Fatalf("ReadSet should dedupe by tag; got Len=%d", r.ReadSet().Len())
		}
	})
}
