// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
)

func makePackage() *emit.Package {
	target := emit.Target{Dir: "internal/users", Filename: "user_gen.go", Package: "users"}
	file := &emit.File{Name: "user_gen.go", Package: "users", Dir: "internal/users"}
	return &emit.Package{
		Name: "users",
		Path: "github.com/example/users",
		Dir:  "internal/users",
		Files: []*emit.File{
			file,
			{Name: "other.go", Package: "users", Dir: "internal/users"},
		},
		Imports: []*emit.Import{
			{Path: "context"},
		},
		Structs:    []*emit.Struct{{Name: "User", Target: target}},
		Interfaces: []*emit.Interface{{Name: "Repo"}},
		Functions:  []*emit.Function{{Name: "Open"}},
		Variables:  []*emit.Variable{{Name: "Default"}},
		Constants:  []*emit.Constant{{Name: "Pi"}},
		Enums:      []*emit.Enum{{Name: "Status"}},
		Aliases:    []*emit.Alias{{Name: "ID"}},
	}
}

func TestPackage_Kind(t *testing.T) {
	t.Parallel()

	t.Run("reports KindPackage", func(t *testing.T) {
		t.Parallel()
		var p emit.Package
		if p.Kind() != emit.KindPackage {
			t.Fatalf("Kind = %s, want %s", p.Kind(), emit.KindPackage)
		}
	})
}

func TestPackage_Slot(t *testing.T) {
	t.Parallel()

	t.Run("named slot lookup is idempotent", func(t *testing.T) {
		t.Parallel()
		p := makePackage()
		if a, b := p.Slot("custom"), p.Slot("custom"); a != b {
			t.Fatalf("Slot lookup should be idempotent")
		}
	})
}

func TestPackage_FileByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching file", func(t *testing.T) {
		t.Parallel()
		got := makePackage().FileByName("user_gen.go")
		if got == nil || got.Name != "user_gen.go" {
			t.Fatalf("FileByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makePackage().FileByName("missing.go") != nil {
			t.Fatalf("FileByName(unknown) should be nil")
		}
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		t.Parallel()
		if makePackage().FileByName("") != nil {
			t.Fatalf("FileByName(\"\") should be nil")
		}
	})
}

func TestPackage_FileByTarget(t *testing.T) {
	t.Parallel()

	t.Run("returns the file matching the supplied Target", func(t *testing.T) {
		t.Parallel()
		want := emit.Target{Dir: "internal/users", Filename: "user_gen.go", Package: "users"}
		got := makePackage().FileByTarget(want)
		if got == nil || got.Name != "user_gen.go" {
			t.Fatalf("FileByTarget mismatch: %+v", got)
		}
	})

	t.Run("returns nil when no file matches the Target", func(t *testing.T) {
		t.Parallel()
		got := makePackage().FileByTarget(emit.Target{Dir: "missing", Filename: "x.go"})
		if got != nil {
			t.Fatalf("FileByTarget should be nil for unknown Target; got %+v", got)
		}
	})
}

func TestPackage_ImportByPath(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching import", func(t *testing.T) {
		t.Parallel()
		got := makePackage().ImportByPath("context")
		if got == nil || got.Path != "context" {
			t.Fatalf("ImportByPath mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown path", func(t *testing.T) {
		t.Parallel()
		if makePackage().ImportByPath("missing") != nil {
			t.Fatalf("ImportByPath(unknown) should be nil")
		}
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		t.Parallel()
		if makePackage().ImportByPath("") != nil {
			t.Fatalf("ImportByPath(\"\") should be nil")
		}
	})
}

func TestPackage_StructByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching struct", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().StructByName("User"); got == nil || got.Name != "User" {
			t.Fatalf("StructByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makePackage().StructByName("missing") != nil {
			t.Fatalf("StructByName(unknown) should be nil")
		}
	})
}

func TestPackage_InterfaceByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching interface", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().InterfaceByName("Repo"); got == nil || got.Name != "Repo" {
			t.Fatalf("InterfaceByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makePackage().InterfaceByName("missing") != nil {
			t.Fatalf("InterfaceByName(unknown) should be nil")
		}
	})
}

func TestPackage_FunctionByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching function", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().FunctionByName("Open"); got == nil || got.Name != "Open" {
			t.Fatalf("FunctionByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makePackage().FunctionByName("missing") != nil {
			t.Fatalf("FunctionByName(unknown) should be nil")
		}
	})
}

func TestPackage_VariableByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching variable", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().VariableByName("Default"); got == nil || got.Name != "Default" {
			t.Fatalf("VariableByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makePackage().VariableByName("missing") != nil {
			t.Fatalf("VariableByName(unknown) should be nil")
		}
	})
}

func TestPackage_ConstantByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching constant", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().ConstantByName("Pi"); got == nil || got.Name != "Pi" {
			t.Fatalf("ConstantByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makePackage().ConstantByName("missing") != nil {
			t.Fatalf("ConstantByName(unknown) should be nil")
		}
	})
}

func TestPackage_EnumByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching enum", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().EnumByName("Status"); got == nil || got.Name != "Status" {
			t.Fatalf("EnumByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makePackage().EnumByName("missing") != nil {
			t.Fatalf("EnumByName(unknown) should be nil")
		}
	})
}

func TestPackage_AliasByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching alias", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().AliasByName("ID"); got == nil || got.Name != "ID" {
			t.Fatalf("AliasByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if makePackage().AliasByName("missing") != nil {
			t.Fatalf("AliasByName(unknown) should be nil")
		}
	})
}

func TestPackage_FilesByTarget(t *testing.T) {
	t.Parallel()

	t.Run("filters files by predicate", func(t *testing.T) {
		t.Parallel()
		got := makePackage().FilesByTarget(func(f *emit.File) bool { return f.Name == "user_gen.go" })
		if len(got) != 1 || got[0].Name != "user_gen.go" {
			t.Fatalf("FilesByTarget mismatch: %+v", got)
		}
	})
}
