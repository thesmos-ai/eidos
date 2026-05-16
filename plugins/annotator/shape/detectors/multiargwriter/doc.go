// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package multiargwriter recognises a writer with three or more
// non-context parameters — `(ctx, a, b, c, ...) error`. Common
// shape for "record N facts" or "apply N changes" entrypoints.
//
// The recognised Go signature is:
//
//	func (r *Repo) Record(ctx context.Context, a A, b B, c C, ...) error
//
// A positive detection stamps:
//
//	shape                          = "multiargwriter"
//	shape.multiargwriter.arg_types = qualified types of every non-context parameter
//
// No structural key_type / value_type is stamped — the shape
// carries N>=3 distinct positional arguments and the per-shape
// list is the canonical accessor.
package multiargwriter
