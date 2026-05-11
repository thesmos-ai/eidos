// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package sink defines the [Sink] interface a backend writes
// rendered output through. The interface is intentionally minimal —
// concrete sinks (disk, stdout, in-memory, multi) implement it to
// route the backend's per-target output to the right destination.
//
// The interface itself is the only contract this package exposes;
// implementations live in subpackages.
package sink
