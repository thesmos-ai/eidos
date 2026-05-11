// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta_test

import (
	"slices"
	"sync"
	"testing"

	"go.thesmos.sh/eidos/core/meta"
)

// observerTestKey is a package-level meta.Key declared once so the
// global key registry never sees re-registration on -count > 1.
var observerTestKey = meta.NewKey("meta.observer.test", meta.BoolParser)

func TestBag_AddObserver(t *testing.T) {
	t.Parallel()

	t.Run("observer fires on Set with the key name", func(t *testing.T) {
		t.Parallel()
		bag := meta.NewBag()
		var seen []string
		bag.AddObserver(func(name string) {
			seen = append(seen, name)
		})
		observerTestKey.Set(bag, true, "test")
		if !slices.Equal(seen, []string{observerTestKey.Name()}) {
			t.Fatalf("observer history = %v, want [%s]", seen, observerTestKey.Name())
		}
	})

	t.Run("multiple observers fire in registration order", func(t *testing.T) {
		t.Parallel()
		bag := meta.NewBag()
		var order []int
		bag.AddObserver(func(string) { order = append(order, 1) })
		bag.AddObserver(func(string) { order = append(order, 2) })
		bag.AddObserver(func(string) { order = append(order, 3) })
		observerTestKey.Set(bag, true, "test")
		if !slices.Equal(order, []int{1, 2, 3}) {
			t.Fatalf("observer order = %v, want [1 2 3]", order)
		}
	})

	t.Run("tombstones do not fire observers", func(t *testing.T) {
		t.Parallel()
		bag := meta.NewBag()
		var seen int
		bag.AddObserver(func(string) { seen++ })
		observerTestKey.Tombstone(bag, "test")
		if seen != 0 {
			t.Fatalf("Tombstone should not fire observers; got %d", seen)
		}
	})

	t.Run("AddObserver and Set are safe under -race", func(t *testing.T) {
		t.Parallel()
		bag := meta.NewBag()
		var counter int
		var mu sync.Mutex
		bag.AddObserver(func(string) {
			mu.Lock()
			counter++
			mu.Unlock()
		})
		var wg sync.WaitGroup
		for range 16 {
			wg.Go(func() {
				observerTestKey.Set(bag, true, "test")
			})
		}
		wg.Wait()
		if counter != 16 {
			t.Fatalf("counter = %d, want 16", counter)
		}
	})
}
