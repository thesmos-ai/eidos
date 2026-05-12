// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package builder

import "go.thesmos.sh/eidos/emit"

// This file holds the value-type constructors — small one-liner
// helpers for emit value types ([emit.Tag], [emit.UnionTerm],
// [emit.Constraint]) that flow across builder API boundaries.
// Plugin authors construct a value here, then pass it to a
// consumer (slot append, Union, TypeParam, etc.).
//
// Wrappers carry no behaviour beyond forwarding; semantics live
// in [emit]. The motivation is one-stop-shop discoverability and
// shorter call sites than direct struct literals.

// Tag returns a freshly-allocated [emit.Tag] with the supplied
// key and value. Used as the cross-cutting struct-tag entry
// flowing into [Context.AppendTag] / [Context.InsertTag].
//
//	c.AppendTag(field, builder.Tag("json", "name,omitempty"))
func Tag(key, value string) *emit.Tag { return &emit.Tag{Key: key, Value: value} }

// Approx returns an [emit.UnionTerm] marked as approximate — Go's
// `~T` constraint-union term. Used as a constructor for the
// terms passed into [emit.Union]:
//
//	emit.Union(builder.Approx(emit.Builtin("int")), builder.Exact(emit.Builtin("string")))
func Approx(ref emit.Ref) emit.UnionTerm { return emit.UnionTerm{Type: ref, Approx: true} }

// Exact returns an [emit.UnionTerm] for an exact-type constraint
// — the `T` (no tilde) form. Mirror of [Approx]; supplied for
// API symmetry so call sites read uniformly even when no `~`
// terms are present.
func Exact(ref emit.Ref) emit.UnionTerm { return emit.UnionTerm{Type: ref} }

// AnyConstraint returns the "implicit-any" constraint — a nil
// [*emit.Constraint], which [emit.Constraint.IsAny] interprets as
// the predeclared `any` bound. Provided so call sites express
// intent explicitly rather than passing a bare nil.
//
//	sb.TypeParam("T", builder.AnyConstraint())
func AnyConstraint() *emit.Constraint { return nil }

// ComparableConstraint returns a constraint bound to the
// predeclared `comparable` identifier — the canonical bound for
// generics that need equality.
//
//	sb.TypeParam("K", builder.ComparableConstraint())
func ComparableConstraint() *emit.Constraint {
	return &emit.Constraint{Embedded: []emit.Ref{emit.Builtin("comparable")}}
}

// Constraint returns a constraint whose [emit.Constraint.Embedded]
// list contains refs. Variadic for ergonomic call sites:
//
//	sb.TypeParam("T", builder.Constraint(emit.Builtin("comparable"), emit.External("fmt", "Stringer")))
//
// Passing zero refs produces the implicit-any case ([AnyConstraint]).
func Constraint(refs ...emit.Ref) *emit.Constraint {
	if len(refs) == 0 {
		return nil
	}
	return &emit.Constraint{Embedded: refs}
}
