// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
	"go.thesmos.sh/eidos/node"
)

// TestContext_ForBindsIdentityAndTarget covers the foundational
// [builder.For] constructor: the Context exposes the bound plugin
// identity and target unchanged, and provenance helpers stamp the
// same plugin name.
func TestContext_ForBindsIdentityAndTarget(t *testing.T) {
	t.Parallel()

	t.Run("SetBy and Target return the constructor arguments", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		if got := c.SetBy(); got != "repogen" {
			t.Fatalf("SetBy = %q, want %q", got, "repogen")
		}
		if got := c.Target(); got != defaultTarget {
			t.Fatalf("Target = %v, want %v", got, defaultTarget)
		}
	})

	t.Run("Provenance stamps SetBy and (optionally) ID", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		bare := c.Provenance()
		if bare.SetBy != "repogen" || bare.ID != "" {
			t.Fatalf("Provenance() = %+v; want SetBy=repogen ID=empty", bare)
		}
		withID := c.Provenance("seed")
		if withID.SetBy != "repogen" || withID.ID != "seed" {
			t.Fatalf("Provenance(\"seed\") = %+v; want SetBy=repogen ID=seed", withID)
		}
	})

	t.Run("WithTarget returns a copy without mutating the receiver", func(t *testing.T) {
		t.Parallel()
		c := builder.For("repogen", defaultTarget)
		other := emit.Target{Dir: "audit", Filename: "audit.go", Package: "users"}
		c2 := c.WithTarget(other)
		if c.Target() != defaultTarget {
			t.Fatalf("receiver should be untouched; got %v", c.Target())
		}
		if c2.Target() != other {
			t.Fatalf("clone should carry the new target; got %v", c2.Target())
		}
	})
}

// TestContext_For_TargetOptional pins the variadic [emit.Target]
// argument: passing no target leaves the Context's default at the
// zero value. Production plugins call [builder.For] without a
// target so the pipeline's Layout phase composes placement.
func TestContext_For_TargetOptional(t *testing.T) {
	t.Parallel()

	t.Run("no target leaves Target at the zero value", func(t *testing.T) {
		t.Parallel()
		c := builder.For("p")
		if got := c.Target(); got != (emit.Target{}) {
			t.Fatalf("Target = %v, want zero", got)
		}
	})
}

// TestContext_Anchor covers the declarative-anchor entry point. The
// Anchor accepts a source node, derives the [emit.Package.Path]
// from it, and stamps the node as the default [emit.BaseEmit.OriginNode]
// on every decl built through the resulting PackageBuilder.
func TestContext_Anchor(t *testing.T) {
	t.Parallel()

	t.Run("Path derives from the anchor node's package", func(t *testing.T) {
		t.Parallel()
		iface := &node.Interface{Name: "Store", Package: "example.com/x/store"}
		pkg := builder.For("mg").Anchor(iface).Node()
		if got, want := pkg.Path, "example.com/x/store"; got != want {
			t.Fatalf("Package.Path = %q, want %q", got, want)
		}
		if got := pkg.Name; got != "" {
			t.Fatalf("Package.Name = %q, want empty (framework fills)", got)
		}
	})

	t.Run("default Origin stamps on decls built via the PackageBuilder", func(t *testing.T) {
		t.Parallel()
		iface := &node.Interface{Name: "Store", Package: "example.com/x/store"}
		var s *emit.Struct
		builder.For("mg").Anchor(iface).Struct("StoreMock", func(sb *builder.StructBuilder) {
			s = sb.Node()
		})
		if s.Origin() != iface {
			t.Fatalf("default Origin = %v, want %v", s.Origin(), iface)
		}
	})

	t.Run("per-decl Origin override wins over the anchor default", func(t *testing.T) {
		t.Parallel()
		anchor := &node.Interface{Name: "A", Package: "x"}
		override := &node.Struct{Name: "Other"}
		var s *emit.Struct
		builder.For("mg").Anchor(anchor).Struct("SMock", func(sb *builder.StructBuilder) {
			sb.Origin(override)
			s = sb.Node()
		})
		if s.Origin() != override {
			t.Fatalf("override Origin = %v, want %v", s.Origin(), override)
		}
	})
}
