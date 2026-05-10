// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package store is the indexed in-memory database the pipeline uses
// to share nodes and emit between phases. A [Store] holds two views:
// [NodeView] for source-side [node.Node] values produced by frontends,
// and [EmitView] for [emit.Node] values produced by generators.
// Plugins query the store through typed accessors that resolve in
// O(matches), never O(N).
//
// The store maintains automatic indices on every recorded entry: by
// concrete kind, by qualified name, by declaring package, and by
// directive presence. Indices update synchronously inside the
// recording call. Iteration is in stable insertion order — frontends
// are expected to insert in source order so downstream phases see
// deterministic output.
//
// All accessors are safe for concurrent use; reads use a shared lock,
// writes take exclusive. The pipeline's mutability contract (frontend
// writes during the frontend phase, annotators set metadata during
// the annotator phase, generators write emit during the generator
// phase) keeps contention low in practice.
//
// The [ReadSet] primitive records what a plugin observed during a
// run; later milestones use the captured reads as the cache key for
// the plugin's outputs.
package store
