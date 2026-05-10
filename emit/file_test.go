// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func makeFile() *emit.File {
	return &emit.File{
		Name:    "user_gen.go",
		Package: "users",
		Dir:     "internal/users",
		Imports: []*emit.Import{
			{Path: "context"},
			{Path: "fmt", Alias: "f"},
		},
	}
}

func TestFile_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindFile", func(t *testing.T) {
		t.Parallel()
		var f emit.File
		if f.Kind() != emit.KindFile {
			t.Fatalf("Kind = %s, want %s", f.Kind(), emit.KindFile)
		}
	})
}

func TestFile_Target(t *testing.T) {
	t.Parallel()

	t.Run("composes Dir, Name, and Package into a Target", func(t *testing.T) {
		t.Parallel()
		got := makeFile().Target()
		want := emit.Target{Dir: "internal/users", Filename: "user_gen.go", Package: "users"}
		if got != want {
			t.Fatalf("Target = %+v, want %+v", got, want)
		}
	})
}

func TestFile_Path(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		f    *emit.File
		want string
	}{
		{"both populated", makeFile(), "internal/users/user_gen.go"},
		{"empty Dir yields empty", &emit.File{Name: "user.go"}, ""},
		{"empty Name yields empty", &emit.File{Dir: "internal/users"}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertEqualString(t, tc.f.Path(), tc.want)
		})
	}
}

func TestFile_Slots(t *testing.T) {
	t.Parallel()

	t.Run("Top, Bottom, Init, ImportsSlot, Slot are distinct and idempotent", func(t *testing.T) {
		t.Parallel()
		f := makeFile()
		t1, t2 := f.Top(), f.Top()
		b1, b2 := f.Bottom(), f.Bottom()
		i1, i2 := f.Init(), f.Init()
		is1, is2 := f.ImportsSlot(), f.ImportsSlot()
		c1, c2 := f.Slot("custom"), f.Slot("custom")
		if t1 != t2 || b1 != b2 || i1 != i2 || is1 != is2 || c1 != c2 {
			t.Fatalf("slot lookups should be idempotent")
		}
		if t1 == b1 || b1 == i1 || i1 == is1 {
			t.Fatalf("standard slots must be distinct instances")
		}
	})
}

func TestFile_ImportByPath(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching import", func(t *testing.T) {
		t.Parallel()
		got := makeFile().ImportByPath("context")
		if got == nil || got.Path != "context" {
			t.Fatalf("ImportByPath mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown path", func(t *testing.T) {
		t.Parallel()
		if makeFile().ImportByPath("missing") != nil {
			t.Fatalf("ImportByPath(unknown) should be nil")
		}
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		t.Parallel()
		if makeFile().ImportByPath("") != nil {
			t.Fatalf("ImportByPath(\"\") should be nil")
		}
	})
}

func TestFile_ImportByAlias(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching import", func(t *testing.T) {
		t.Parallel()
		got := makeFile().ImportByAlias("f")
		if got == nil || got.Alias != "f" {
			t.Fatalf("ImportByAlias mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown alias", func(t *testing.T) {
		t.Parallel()
		if makeFile().ImportByAlias("missing") != nil {
			t.Fatalf("ImportByAlias(unknown) should be nil")
		}
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		t.Parallel()
		if makeFile().ImportByAlias("") != nil {
			t.Fatalf("ImportByAlias(\"\") should be nil")
		}
	})
}
