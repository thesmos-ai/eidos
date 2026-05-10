// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"testing"

	"go.thesmos.sh/eidos/store"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns a Store with non-nil Nodes and Emit views", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		if s.Nodes() == nil {
			t.Fatalf("Nodes view should be non-nil")
		}
		if s.Emit() == nil {
			t.Fatalf("Emit view should be non-nil")
		}
	})
}

func TestStore_Nodes(t *testing.T) {
	t.Parallel()

	t.Run("returns the same NodeView on every call", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		if a, b := s.Nodes(), s.Nodes(); a != b {
			t.Fatalf("Nodes() should return the cached view")
		}
	})
}

func TestStore_Emit(t *testing.T) {
	t.Parallel()

	t.Run("returns the same EmitView on every call", func(t *testing.T) {
		t.Parallel()
		s := store.New()
		if a, b := s.Emit(), s.Emit(); a != b {
			t.Fatalf("Emit() should return the cached view")
		}
	})
}
