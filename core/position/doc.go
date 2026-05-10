// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package position carries source location information end-to-end.
//
// Pos identifies a single byte location in a source file; Range spans
// two such locations. Both are value types and safe to copy.
//
// Real source positions reference paths to actual files, populated by
// frontends as they parse. Synthetic positions for purely-generated
// artifacts use opaque markers (the convention is "<generated>" in
// File) so consumers can distinguish them from real source.
//
// The zero value of Pos and Range represents "no position is known".
// Diagnostics, provenance trails, and explain output treat such
// positions as missing rather than as a valid origin.
package position
