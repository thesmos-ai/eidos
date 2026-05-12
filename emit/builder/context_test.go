// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/emit/builder"
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
