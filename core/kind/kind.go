// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package kind defines the [Kind] string-aliased type that
// discriminates structural categories on the source-side
// [node.Node] graph and the emit-side [emit.Node] graph, and
// that [directive.Schema.AppliesTo] uses to scope a directive
// to specific node kinds.
//
// The type lives in its own package to keep [directive],
// [node], and [emit] decoupled. Each owns its own [Kind]
// constants — [node.KindStruct] for source structs,
// [emit.KindStruct] for emit structs, plugin-defined kinds for
// any user-introduced emit shapes — but the underlying type is
// shared so directive schemas can reference any kind without
// circular imports.
package kind

// Kind is the symbolic name of a structural category on a
// [node.Node] or [emit.Node] ("struct", "interface", "method",
// and so on).
//
// Defined as a bare string-aliased type so concrete kind
// constants live in their owning package
// ([node.KindStruct], [emit.KindStruct], plugin-defined names
// for plugin-introduced emit kinds) and so
// [directive.Schema.AppliesTo] is self-describing at call
// sites — `directive.NewSchema("repo").On(node.KindStruct)`
// reads as English.
type Kind string
