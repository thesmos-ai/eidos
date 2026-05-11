// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package sink

import "errors"

// ErrInvalidTarget is returned by [Sink.Write] when the supplied
// [emit.Target] is missing the [emit.Target.Filename] component the
// sink needs to determine the destination. Empty-target writes are
// always programmer errors — the backend builds Targets from real
// emit entities and the values are required to be populated by the
// time they reach a sink.
var ErrInvalidTarget = errors.New("sink: invalid target")
