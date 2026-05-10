// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package meta is the universal extension mechanism for nodes and
// emit entities. It stores typed key/value facts (and tombstones)
// alongside a record of who set what, at which [Authority], and
// where.
//
// Keys are typed: a plugin declares a [Key][T] once, then reads and
// writes it with full compile-time type safety. The same key
// participates in three storage modes — plugin, directive, and
// manual — each ranked higher than the previous. Resolution
// (via [Key.Get] or [Bag.Winning]) always returns the value from the
// highest authority; tombstones win ties within an authority.
//
// Prefix tombstones cascade: tombstoning "shape.writer" at
// [AuthorityDirective] suppresses every "shape.writer.*" descendant
// at the same authority, regardless of whether those descendants
// have been set yet. This is what makes a directive like
// "-gen:shape writer" scrub the entire writer-shape fact bundle in
// one stroke.
//
// The directive-override step uses [AnyKey] to resolve a string key
// name into the registered typed parser and stamp the parsed value
// into a [Bag] at [AuthorityDirective]. Plugins never have to do
// untyped access themselves; the typed API on [Key][T] covers every
// plugin use case.
//
// [Provenance] records every recorded operation per name. Tooling
// such as `eidos explain` walks the history to render the full
// "why is this stamped" story.
package meta
