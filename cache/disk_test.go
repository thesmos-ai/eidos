// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package cache_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go.thesmos.sh/eidos/cache"
)

func TestNewDisk(t *testing.T) {
	t.Parallel()

	t.Run("captures the supplied root", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk("/tmp/eidos-cache")
		if d.Root() != "/tmp/eidos-cache" {
			t.Fatalf("Root = %q, want /tmp/eidos-cache", d.Root())
		}
	})
}

func TestDisk_PutAndGet(t *testing.T) {
	t.Parallel()

	t.Run("Put-then-Get round-trips bytes", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		key := "abcdef0123456789"
		assertNoError(t, d.Put(key, []byte("hello")))
		got, ok := d.Get(key)
		if !ok || string(got) != "hello" {
			t.Fatalf("round-trip mismatch: %q ok=%v", got, ok)
		}
	})

	t.Run("Put with a short key (< 2 chars) stores under root directly", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		d := cache.NewDisk(root)
		assertNoError(t, d.Put("a", []byte("body")))
		// Short keys go directly under root with the key as the file
		// name; verify by reading the file back.
		got, ok := d.Get("a")
		if !ok || string(got) != "body" {
			t.Fatalf("short-key round-trip mismatch: %q ok=%v", got, ok)
		}
	})

	t.Run("Put with the same key replaces the prior body", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		key := "abcdef0123"
		assertNoError(t, d.Put(key, []byte("first")))
		assertNoError(t, d.Put(key, []byte("second")))
		got, _ := d.Get(key)
		if string(got) != "second" {
			t.Fatalf("Put should replace; got %q", got)
		}
	})
}

func TestDisk_Get(t *testing.T) {
	t.Parallel()

	t.Run("Get on missing key returns nil and false", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		if got, ok := d.Get("missing"); ok || got != nil {
			t.Fatalf("Get on missing key = %q ok=%v", got, ok)
		}
	})

	t.Run("Get with empty key returns a miss", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		if _, ok := d.Get(""); ok {
			t.Fatalf("Get with empty key should be a miss")
		}
	})

	t.Run("Get on an unreadable entry reports a miss", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		d := cache.NewDisk(root)
		key := "abcdef0123"
		assertNoError(t, d.Put(key, []byte("body")))
		// Make the per-key directory unreadable so the subsequent
		// Get hits the read error path rather than finding the file.
		bucket := filepath.Join(root, key[:2])
		assertNoError(t, os.Chmod(bucket, 0o000))         //nolint:gosec // intentional unreadable
		t.Cleanup(func() { _ = os.Chmod(bucket, 0o750) }) //nolint:gosec // restore for TempDir cleanup
		if _, ok := d.Get(key); ok {
			t.Fatalf("Get on unreadable entry should report a miss")
		}
	})
}

func TestDisk_Put(t *testing.T) {
	t.Parallel()

	t.Run("rejects empty key with ErrInvalidKey", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		err := d.Put("", []byte("body"))
		if !errors.Is(err, cache.ErrInvalidKey) {
			t.Fatalf("Put with empty key should return ErrInvalidKey; got %v", err)
		}
	})

	t.Run("returns an error when MkdirAll fails", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		// Pre-create a regular file at the path the Disk wants to
		// use as the bucket directory.
		conflictPath := filepath.Join(root, "ab")
		assertNoError(t, os.WriteFile(conflictPath, nil, 0o600))
		d := cache.NewDisk(root)
		if err := d.Put("abcdef", []byte("body")); err == nil {
			t.Fatalf("Put should fail when MkdirAll cannot create the bucket dir")
		}
	})

	t.Run("returns an error when WriteFile fails on a read-only bucket", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		bucket := filepath.Join(root, "ab")
		assertNoError(t, os.MkdirAll(bucket, 0o750))
		assertNoError(t, os.Chmod(bucket, 0o500))         //nolint:gosec // intentional read-only
		t.Cleanup(func() { _ = os.Chmod(bucket, 0o750) }) //nolint:gosec // restore for TempDir cleanup
		d := cache.NewDisk(root)
		if err := d.Put("abcdef", []byte("body")); err == nil {
			t.Fatalf("Put should fail when WriteFile cannot create a file in a read-only bucket")
		}
	})

	t.Run("returns an error when Rename targets an existing directory", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		// Pre-create the destination as a directory so Rename can't
		// replace it with a file.
		assertNoError(t, os.MkdirAll(filepath.Join(root, "ab", "abcdef"), 0o750))
		d := cache.NewDisk(root)
		if err := d.Put("abcdef", []byte("body")); err == nil {
			t.Fatalf("Put should fail when Rename collides with an existing directory")
		}
	})
}

func TestDisk_Has(t *testing.T) {
	t.Parallel()

	t.Run("reports true for previously-put keys", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		assertNoError(t, d.Put("abcdef", []byte("body")))
		if !d.Has("abcdef") {
			t.Fatalf("Has should return true after Put")
		}
	})

	t.Run("reports false for missing keys", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		if d.Has("nope") {
			t.Fatalf("Has on missing key should return false")
		}
	})

	t.Run("reports false for empty keys", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		if d.Has("") {
			t.Fatalf("Has on empty key should return false")
		}
	})
}

func TestDisk_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("Put and Get are safe under -race", func(t *testing.T) {
		t.Parallel()
		d := cache.NewDisk(t.TempDir())
		var wg sync.WaitGroup
		for i := range 16 {
			wg.Go(func() {
				key := "abc" + string(rune('a'+i%26))
				_ = d.Put(key, []byte("body"))
			})
		}
		for range 4 {
			wg.Go(func() {
				_, _ = d.Get("abca")
			})
		}
		wg.Wait()
	})
}

func TestDisk_SatisfiesCache(t *testing.T) {
	t.Parallel()

	t.Run("Disk satisfies the Cache interface", func(t *testing.T) {
		t.Parallel()
		var _ cache.Cache = cache.NewDisk("/tmp/eidos-test")
	})
}
