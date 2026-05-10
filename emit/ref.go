// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package emit

// Ref is the marker interface every type reference satisfies. The
// concrete reference types — [TypeRef] for internal refs to other
// emit entities, [ExternalRef] for third-party / stdlib types,
// [BuiltinRef] for language built-ins, and [CompositeRef] for
// compound shapes (pointer, slice, array, map, func, chan) over
// inner refs — embed [BaseEmit] and report their kind via [Node.Kind].
//
// Methods receiving a Ref typically run a type switch over the four
// implementations; the [Node] superset method set is available on
// every Ref for shared concerns (position, docs, directives, meta,
// origin).
type Ref interface {
	Node
	isRef()
}
