// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"errors"
	"fmt"
)

// ErrDuplicateEntity is the umbrella sentinel for cross-plugin or
// cross-generator duplicate declarations and slot entries. Two
// emit values sharing a discriminator (qualified name, slot key,
// …) that would collide in the rendered output surface this
// sentinel — directly or via a more specific wrap such as
// [ErrDuplicateQName].
//
// Backends and tooling match against this sentinel when they need
// to recognise "two things wanted to be the one thing" regardless
// of which layer detected the collision.
var ErrDuplicateEntity = errors.New("duplicate entity")

// ErrDuplicateQName is returned when an Add* call attempts to record
// a value whose qualified name collides with one already present in
// the store. Frontends and generators must not produce duplicate
// declarations within a single run; collisions surface as bugs in
// the producer or as a missing dedup step earlier in the pipeline.
//
// ErrDuplicateQName wraps [ErrDuplicateEntity] so consumers that
// only care about the umbrella case can match the broader
// sentinel via [errors.Is].
var ErrDuplicateQName = fmt.Errorf("%w: store: duplicate qualified name", ErrDuplicateEntity)

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
