// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store

import "errors"

// ErrDuplicateQName is returned when an Add* call attempts to record
// a value whose qualified name collides with one already present in
// the store. Frontends and generators must not produce duplicate
// declarations within a single run; collisions surface as bugs in
// the producer or as a missing dedup step earlier in the pipeline.
var ErrDuplicateQName = errors.New("store: duplicate qualified name")

// ErrNilEntry is returned when an Add* call is invoked with a nil
// pointer. Callers must produce non-nil entries before recording
// them; nil entries break index invariants and downstream traversal.
var ErrNilEntry = errors.New("store: nil entry")

// ErrFrozen is returned by an Add* call after the view has been
// frozen by [NodeView.Freeze] or [EmitView.Freeze]. The pipeline
// freezes views between phases to enforce the mutability contract;
// a plugin that tries to mutate a frozen view has violated its
// phase's read-only invariant.
//
// Pipelines surface this as an [diag.Internal] diagnostic because
// the violation indicates a framework-contract bug in the plugin,
// not a problem with the user's source code.
var ErrFrozen = errors.New("store: view is frozen")
