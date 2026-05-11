// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cache_test

import (
	"testing"

	"go.thesmos.sh/eidos/cache"
)

func TestNewNone(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil None cache", func(t *testing.T) {
		t.Parallel()
		if cache.NewNone() == nil {
			t.Fatalf("NewNone should return a non-nil instance")
		}
	})
}

func TestNone_Get(t *testing.T) {
	t.Parallel()

	t.Run("always reports a miss", func(t *testing.T) {
		t.Parallel()
		c := cache.NewNone()
		if got, ok := c.Get("k"); ok || got != nil {
			t.Fatalf("Get on None should always miss; got %q ok=%v", got, ok)
		}
	})
}

func TestNone_Put(t *testing.T) {
	t.Parallel()

	t.Run("always returns nil and discards the value", func(t *testing.T) {
		t.Parallel()
		c := cache.NewNone()
		assertNoError(t, c.Put("k", []byte("body")))
		if _, ok := c.Get("k"); ok {
			t.Fatalf("None should not retain put values")
		}
	})
}

func TestNone_SatisfiesCache(t *testing.T) {
	t.Parallel()

	t.Run("None satisfies the Cache interface", func(t *testing.T) {
		t.Parallel()
		var _ cache.Cache = cache.NewNone()
	})
}
