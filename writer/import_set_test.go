// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package writer_test

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"go.thesmos.sh/eidos/writer"
)

func TestDefaultAlias(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		want string
	}{
		{"single-segment path", "context", "context"},
		{"multi-segment path", "github.com/foo/bar", "bar"},
		{"trailing slash returns empty", "github.com/foo/", ""},
		{"empty path returns empty", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := writer.DefaultAlias(tc.path); got != tc.want {
				t.Fatalf("DefaultAlias(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestNewImportSet(t *testing.T) {
	t.Parallel()

	t.Run("returns an empty ImportSet ready for use", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		if is.Len() != 0 {
			t.Fatalf("new ImportSet should be empty")
		}
	})

	t.Run("nil derive function defaults to DefaultAlias", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		alias, err := is.Imp("github.com/foo/bar")
		assertNoError(t, err)
		if alias != "bar" {
			t.Fatalf("default derivation should pick last segment; got %q", alias)
		}
	})
}

func TestImportSet_Imp(t *testing.T) {
	t.Parallel()

	t.Run("first call returns the derived alias", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		alias, err := is.Imp("context")
		assertNoError(t, err)
		if alias != "context" {
			t.Fatalf("alias = %q, want context", alias)
		}
	})

	t.Run("repeat calls for the same path return the same alias", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		first, err := is.Imp("github.com/foo/bar")
		assertNoError(t, err)
		second, err := is.Imp("github.com/foo/bar")
		assertNoError(t, err)
		if first != second {
			t.Fatalf("repeat Imp returned different aliases: %q vs %q", first, second)
		}
	})

	t.Run("colliding aliases get a numeric suffix deterministically", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		a1, err := is.Imp("context")
		assertNoError(t, err)
		a2, err := is.Imp("github.com/foo/context")
		assertNoError(t, err)
		a3, err := is.Imp("github.com/bar/context")
		assertNoError(t, err)
		if a1 != "context" || a2 != "context2" || a3 != "context3" {
			t.Fatalf("collision aliases mismatch: %q %q %q", a1, a2, a3)
		}
	})

	t.Run("empty path returns ErrEmptyPath", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		if _, err := is.Imp(""); !errors.Is(err, writer.ErrEmptyPath) {
			t.Fatalf("Imp(\"\") should return ErrEmptyPath; got %v", err)
		}
	})

	t.Run("custom derive function controls aliasing", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(func(p string) string { return "x_" + writer.DefaultAlias(p) })
		alias, err := is.Imp("context")
		assertNoError(t, err)
		if alias != "x_context" {
			t.Fatalf("custom derive should drive alias; got %q", alias)
		}
	})
}

func TestImportSet_Alias(t *testing.T) {
	t.Parallel()

	t.Run("explicit alias overrides the derived default", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		assertNoError(t, is.Alias("context", "ctx"))
		alias, err := is.Imp("context")
		assertNoError(t, err)
		if alias != "ctx" {
			t.Fatalf("explicit alias should win; got %q", alias)
		}
	})

	t.Run("Alias after Imp returns ErrAliasAfterImp", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		_, err := is.Imp("context")
		assertNoError(t, err)
		err = is.Alias("context", "ctx")
		if !errors.Is(err, writer.ErrAliasAfterImp) {
			t.Fatalf("Alias after Imp should return ErrAliasAfterImp; got %v", err)
		}
	})

	t.Run("empty path returns ErrEmptyPath", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		if err := is.Alias("", "ctx"); !errors.Is(err, writer.ErrEmptyPath) {
			t.Fatalf("Alias(\"\") should return ErrEmptyPath; got %v", err)
		}
	})
}

func TestImportSet_AliasOf(t *testing.T) {
	t.Parallel()

	t.Run("returns the assigned alias and true for a known path", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		_, err := is.Imp("context")
		assertNoError(t, err)
		alias, ok := is.AliasOf("context")
		if !ok || alias != "context" {
			t.Fatalf("AliasOf mismatch: %q ok=%v", alias, ok)
		}
	})

	t.Run("returns \"\" and false for an unknown path", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		if alias, ok := is.AliasOf("missing"); ok || alias != "" {
			t.Fatalf("AliasOf(unknown) should be (\"\", false); got %q ok=%v", alias, ok)
		}
	})
}

func TestImportSet_Imports(t *testing.T) {
	t.Parallel()

	t.Run("returns imports in insertion order", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		_, _ = is.Imp("c")
		_, _ = is.Imp("a")
		_, _ = is.Imp("b")
		got := is.Imports()
		names := []string{got[0].Path, got[1].Path, got[2].Path}
		if !slices.Equal(names, []string{"c", "a", "b"}) {
			t.Fatalf("insertion order mismatch: %v", names)
		}
	})

	t.Run("returns a defensive copy", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		_, _ = is.Imp("a")
		snap := is.Imports()
		snap[0].Path = "MUTATED"
		fresh := is.Imports()
		if fresh[0].Path != "a" {
			t.Fatalf("Imports should return a defensive copy")
		}
	})
}

func TestImportSet_ConcurrentImp(t *testing.T) {
	t.Parallel()

	t.Run("concurrent Imp calls are safe under -race and produce stable aliases", func(t *testing.T) {
		t.Parallel()
		is := writer.NewImportSet(nil)
		var wg sync.WaitGroup
		for range 16 {
			wg.Go(func() {
				_, _ = is.Imp("context")
			})
		}
		wg.Wait()
		if is.Len() != 1 {
			t.Fatalf("16 concurrent Imp(\"context\") should record one path; got Len=%d", is.Len())
		}
	})
}
