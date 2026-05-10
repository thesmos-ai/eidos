// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"
)

func makePackage() *node.Package {
	return &node.Package{
		Name: "users",
		Path: "github.com/example/users",
		Files: []*node.File{
			{Name: "user.go"},
			{Name: "doc.go"},
		},
		Imports: []*node.Import{
			{Path: "context"},
			{Path: "fmt"},
		},
		Structs:    []*node.Struct{{Name: "User"}, {Name: "Profile"}},
		Interfaces: []*node.Interface{{Name: "Repo"}},
		Functions:  []*node.Function{{Name: "Open"}},
		Variables:  []*node.Variable{{Name: "Default"}},
		Constants:  []*node.Constant{{Name: "Pi"}},
		Enums:      []*node.Enum{{Name: "Status"}},
		Aliases:    []*node.Alias{{Name: "ID"}},
	}
}

func TestPackage_StructByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching struct", func(t *testing.T) {
		t.Parallel()
		got := makePackage().StructByName("Profile")
		if got == nil || got.Name != "Profile" {
			t.Fatalf("StructByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().StructByName("missing"); got != nil {
			t.Fatalf("StructByName(unknown) should be nil; got %+v", got)
		}
	})
}

func TestPackage_InterfaceByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching interface", func(t *testing.T) {
		t.Parallel()
		got := makePackage().InterfaceByName("Repo")
		if got == nil || got.Name != "Repo" {
			t.Fatalf("InterfaceByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().InterfaceByName("missing"); got != nil {
			t.Fatalf("InterfaceByName(unknown) should be nil; got %+v", got)
		}
	})
}

func TestPackage_FunctionByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching function", func(t *testing.T) {
		t.Parallel()
		got := makePackage().FunctionByName("Open")
		if got == nil || got.Name != "Open" {
			t.Fatalf("FunctionByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().FunctionByName("missing"); got != nil {
			t.Fatalf("FunctionByName(unknown) should be nil; got %+v", got)
		}
	})
}

func TestPackage_EnumByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching enum", func(t *testing.T) {
		t.Parallel()
		got := makePackage().EnumByName("Status")
		if got == nil || got.Name != "Status" {
			t.Fatalf("EnumByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().EnumByName("missing"); got != nil {
			t.Fatalf("EnumByName(unknown) should be nil; got %+v", got)
		}
	})
}

func TestPackage_AliasByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching alias", func(t *testing.T) {
		t.Parallel()
		got := makePackage().AliasByName("ID")
		if got == nil || got.Name != "ID" {
			t.Fatalf("AliasByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().AliasByName("missing"); got != nil {
			t.Fatalf("AliasByName(unknown) should be nil; got %+v", got)
		}
	})
}

func TestPackage_VariableByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching variable", func(t *testing.T) {
		t.Parallel()
		got := makePackage().VariableByName("Default")
		if got == nil || got.Name != "Default" {
			t.Fatalf("VariableByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().VariableByName("missing"); got != nil {
			t.Fatalf("VariableByName(unknown) should be nil; got %+v", got)
		}
	})
}

func TestPackage_ConstantByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching constant", func(t *testing.T) {
		t.Parallel()
		got := makePackage().ConstantByName("Pi")
		if got == nil || got.Name != "Pi" {
			t.Fatalf("ConstantByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().ConstantByName("missing"); got != nil {
			t.Fatalf("ConstantByName(unknown) should be nil; got %+v", got)
		}
	})
}

func TestPackage_FileByName(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching file", func(t *testing.T) {
		t.Parallel()
		got := makePackage().FileByName("doc.go")
		if got == nil || got.Name != "doc.go" {
			t.Fatalf("FileByName mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an empty name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().FileByName(""); got != nil {
			t.Fatalf("FileByName(\"\") should return nil; got %+v", got)
		}
	})

	t.Run("returns nil for an unknown name", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().FileByName("missing.go"); got != nil {
			t.Fatalf("FileByName(unknown) should be nil; got %+v", got)
		}
	})
}

func TestPackage_ImportByPath(t *testing.T) {
	t.Parallel()

	t.Run("returns the matching import", func(t *testing.T) {
		t.Parallel()
		got := makePackage().ImportByPath("fmt")
		if got == nil || got.Path != "fmt" {
			t.Fatalf("ImportByPath mismatch: %+v", got)
		}
	})

	t.Run("returns nil for an empty path", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().ImportByPath(""); got != nil {
			t.Fatalf("ImportByPath(\"\") should return nil; got %+v", got)
		}
	})

	t.Run("returns nil for an unknown path", func(t *testing.T) {
		t.Parallel()
		if got := makePackage().ImportByPath("missing"); got != nil {
			t.Fatalf("ImportByPath(unknown) should be nil; got %+v", got)
		}
	})
}
