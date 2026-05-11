// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package node

// Constraint is the structured bound on a generic [TypeParam]. It
// captures the universal shape of a generic constraint across
// languages: the printed source form ([Constraint.Raw]) and the
// named-type bounds the parameter must satisfy
// ([Constraint.Embedded]).
//
// Worked examples:
//
//	"any"           → Constraint{Raw: "any"}
//	"comparable"    → Constraint{Embedded: [comparable]}
//	"fmt.Stringer"  → Constraint{Embedded: [fmt.Stringer]}
//
// Language-specific constraint shapes (Go's `~T | U` type-set form,
// Rust's `where T: Display + Debug` chained bounds, etc.) ride on
// metadata stamped by the originating frontend under the
// `<lang>.*` namespace — e.g. the Go frontend records type-set
// terms via a `go.*` key. This keeps the constraint type itself
// language-agnostic.
//
// A nil *Constraint reads as the [Constraint.IsAny] case — no
// explicit bound was declared on the type parameter.
type Constraint struct {
	// Raw is the printed source form of the constraint. Always
	// populated when constructed by a frontend; empty for synthetic
	// constraints built programmatically (tests, fixtures).
	Raw string `json:"raw,omitempty"`

	// Embedded lists the named refs embedded as bounds (interfaces,
	// language-predeclared bound identifiers like Go's
	// `comparable`). Empty when the constraint is purely a
	// language-specific shape carried via metadata.
	Embedded []*TypeRef `json:"embedded,omitempty"`
}

// IsAny reports whether the constraint accepts every type. A nil
// receiver returns true (no explicit bound = "any"); an explicit
// constraint with no Embedded entry also returns true.
func (c *Constraint) IsAny() bool {
	if c == nil {
		return true
	}
	return len(c.Embedded) == 0
}

// IsComparable reports whether the constraint embeds a named ref
// matching Go's predeclared `comparable` identifier (a Named ref
// with no Package and Name == "comparable"). Other languages that
// reuse the "comparable" name follow the same convention.
// Returns false for a nil receiver.
func (c *Constraint) IsComparable() bool {
	if c == nil {
		return false
	}
	for _, e := range c.Embedded {
		if e != nil && e.TypeKind == TypeRefNamed && e.Package == "" && e.Name == "comparable" {
			return true
		}
	}
	return false
}
