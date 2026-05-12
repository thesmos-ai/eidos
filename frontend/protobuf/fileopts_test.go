// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package protobuf_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/frontend/protobuf"
	"go.thesmos.sh/eidos/node"
)

// TestConvert_StampsFileOptions covers the spec's file-level option
// channel: every file-level option (standard plus custom) lands on
// the producing [node.Package] under `proto.option.<full-name>`.
// Standard options use the option's well-known short name (e.g.
// `go_package`, `deprecated`); custom options use the extension's
// proto FQN. Values carry their natural Go type per the spec's
// value-type table.
func TestConvert_StampsFileOptions(t *testing.T) {
	t.Parallel()

	t.Run("standard go_package option stamps under proto.option.go_package", func(t *testing.T) {
		t.Parallel()
		env := loadFixture(t, "fileoptions", "./...")
		if env.diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", env.diag.Diagnostics())
		}
		pkg := requireSinglePackage(t, env)
		key := meta.EnsureKey(protobuf.MetaOptionPrefix+"go_package", meta.StringParser)
		got, ok := key.Get(pkg.Meta())
		if !ok {
			t.Fatalf("expected proto.option.go_package stamp; not present (bag = %+v)", pkg.Meta())
		}
		const want = "github.com/example/fileoptions"
		if got != want {
			t.Fatalf("proto.option.go_package = %q, want %q", got, want)
		}
	})

	t.Run("standard deprecated option stamps under proto.option.deprecated", func(t *testing.T) {
		t.Parallel()
		env := loadFixture(t, "fileoptions", "./...")
		if env.diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", env.diag.Diagnostics())
		}
		pkg := requireSinglePackage(t, env)
		key := meta.EnsureKey(protobuf.MetaOptionPrefix+"deprecated", meta.BoolParser)
		got, ok := key.Get(pkg.Meta())
		if !ok {
			t.Fatalf("expected proto.option.deprecated stamp; not present (bag = %+v)", pkg.Meta())
		}
		if !got {
			t.Fatalf("proto.option.deprecated = false, want true")
		}
	})

	t.Run("custom string-typed extension stamps under its proto FQN", func(t *testing.T) {
		t.Parallel()
		env := loadFixture(t, "fileoptions", "./...")
		if env.diag.HasErrors() {
			t.Fatalf("expected no error diagnostics; got %+v", env.diag.Diagnostics())
		}
		pkg := requireSinglePackage(t, env)
		const name = "eidos.protobuf.testdata.fileoptions.audience"
		key := meta.EnsureKey(protobuf.MetaOptionPrefix+name, meta.StringParser)
		got, ok := key.Get(pkg.Meta())
		if !ok {
			t.Fatalf("expected proto.option.%s stamp; not present (bag = %+v)", name, pkg.Meta())
		}
		const want = "public"
		if got != want {
			t.Fatalf("proto.option.%s = %q, want %q", name, got, want)
		}
	})
}

// requireSinglePackage asserts that exactly one [node.Package]
// landed under env's NodeView and returns it. Used by tests whose
// fixtures emit a single package — the helper centralises the
// shape assertion so each test focuses on its option-specific
// readback.
func requireSinglePackage(t *testing.T, env fixtureEnv) *node.Package {
	t.Helper()
	pkgs := collectPackages(t, env)
	if got := len(pkgs); got != 1 {
		t.Fatalf("expected exactly 1 package; got %d (%+v)", got, packagePaths(pkgs))
	}
	return pkgs[0]
}
