// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package routing_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/routing"
)

// TestAlongsideSource pins the contract: Dir empty (Router fills),
// Filename derived from srcPos's basename + suffix, Package +
// ImportPath inherit from the supplied pkgName / pkgPath.
func TestAlongsideSource(t *testing.T) {
	t.Parallel()

	t.Run("filename combines source basename with suffix; Dir left for Router", func(t *testing.T) {
		t.Parallel()
		target := routing.AlongsideSource(
			position.Pos{File: "internal/users/article.go"}, "Article",
			"users", "example.com/users",
			"_repo.go",
		)
		if target.Dir != "" {
			t.Fatalf("Dir should be empty so the Router fills it; got %q", target.Dir)
		}
		if target.Filename != "article_repo.go" {
			t.Fatalf("Filename = %q, want %q", target.Filename, "article_repo.go")
		}
		if target.Package != "users" {
			t.Fatalf("Package = %q, want %q", target.Package, "users")
		}
		if target.ImportPath != "example.com/users" {
			t.Fatalf("ImportPath = %q, want %q", target.ImportPath, "example.com/users")
		}
	})

	t.Run("filename falls back to srcName when srcPos has no file", func(t *testing.T) {
		t.Parallel()
		target := routing.AlongsideSource(
			position.Pos{}, "Article",
			"users", "example.com/users",
			"_repo.go",
		)
		// srcfile.WithSuffix lowercases the fallback when no Pos is
		// supplied — the rendered filename stays predictable.
		if target.Filename != "article_repo.go" {
			t.Fatalf("Filename fallback = %q, want %q", target.Filename, "article_repo.go")
		}
	})
}

// TestCentralised pins the contract: Filename uses srcName
// lower-cased plus suffix; Dir + Package both equal
// outputPackage; ImportPath empty.
func TestCentralised(t *testing.T) {
	t.Parallel()

	t.Run("Dir + Package match outputPackage; Filename lower-cases srcName", func(t *testing.T) {
		t.Parallel()
		target := routing.Centralised("Article", "gen", "_repo.go")
		if target.Dir != "gen" {
			t.Fatalf("Dir = %q, want %q", target.Dir, "gen")
		}
		if target.Package != "gen" {
			t.Fatalf("Package = %q, want %q", target.Package, "gen")
		}
		if target.Filename != "article_repo.go" {
			t.Fatalf("Filename = %q, want %q", target.Filename, "article_repo.go")
		}
		if target.ImportPath != "" {
			t.Fatalf("ImportPath should be empty for centralised layout; got %q", target.ImportPath)
		}
	})

	t.Run("entity names with mixed case lower-case in the filename", func(t *testing.T) {
		t.Parallel()
		target := routing.Centralised("HTTPHandler", "gen", "_mock.go")
		if target.Filename != "httphandler_mock.go" {
			t.Fatalf("Filename = %q, want %q", target.Filename, "httphandler_mock.go")
		}
	})
}
