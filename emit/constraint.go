// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

// Constraint is the structured bound on a generic [TypeParam]. The
// shape mirrors [node.Constraint] so generators copying a node's
// constraint into their emit graph see a straightforward field-by-
// field mapping with no information loss.
//
// Worked examples:
//
//	"any"           → Constraint{Raw: "any"}
//	"comparable"    → Constraint{Embedded: [BuiltinRef("comparable")]}
//	"fmt.Stringer"  → Constraint{Embedded: [ExternalRef("fmt", "Stringer")]}
//
// Language-specific constraint shapes (Go's `~T | U` type-set form,
// for example) ride on plugin-defined emit kinds or backend-specific
// metadata rather than first-class fields here — emit is a single
// cross-language output model that every backend consumes.
//
// A nil *Constraint reads as the [Constraint.IsAny] case — no
// explicit bound was declared.
type Constraint struct {
	// Raw is the printed source form of the constraint when the
	// constructing generator has it available; empty otherwise —
	// backends fall back to rendering from Embedded in that case.
	Raw string `json:"raw,omitempty"`

	// Embedded lists the named refs embedded as bounds (interfaces,
	// the `comparable` predeclared identifier, etc.). Empty when
	// the constraint is purely a language-specific shape carried
	// via plugin-defined emit kinds.
	Embedded []Ref `json:"-"`
}

// IsAny reports whether the constraint accepts every type. A nil
// receiver returns true; an explicit constraint with no Embedded
// entry also returns true.
func (c *Constraint) IsAny() bool {
	if c == nil {
		return true
	}
	return len(c.Embedded) == 0
}

// IsComparable reports whether the constraint embeds the predeclared
// `comparable` identifier as a [BuiltinRef]. Returns false for a nil
// receiver, for non-builtin embeds named "comparable", and for refs
// of other variants.
func (c *Constraint) IsComparable() bool {
	if c == nil {
		return false
	}
	for _, e := range c.Embedded {
		if b, ok := e.(*BuiltinRef); ok && b != nil && b.Name == "comparable" {
			return true
		}
	}
	return false
}
