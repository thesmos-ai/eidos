// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package codec hosts low-level codec primitives. The package sits
// under `internal/` so it's only importable from within the
// multipkg module — verifying that the frontend handles internal
// import paths without special-casing.
package codec

// Codec is the generic encode/decode surface every transport
// implementation embeds. The two-method shape exercises generic
// interface rendering and the round-trip through refconv when
// callers reference Codec[T] qualified from another package.
//
// +gen:mock
type Codec[T any] interface {
	// Encode marshals value into a byte slice.
	Encode(value T) ([]byte, error)

	// Decode unmarshals raw bytes back into T.
	Decode(raw []byte) (T, error)
}

// PairCodec is a multi-parameter generic that bundles a Key codec
// with a Value codec. The K parameter carries the `comparable`
// constraint to exercise constrained-parameter rendering through
// the storage layer.
type PairCodec[K comparable, V any] struct {
	Key   Codec[K]
	Value Codec[V]
}
