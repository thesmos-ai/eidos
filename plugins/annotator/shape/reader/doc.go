// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package reader recognises the canonical reader shape — a
// callable that takes a key and returns a value plus an error.
//
// The recognised Go signature is:
//
//	func[T any](ctx context.Context, key K) (V, error)
//	func[T any](key K) (V, error)              // ctx-optional
//	func (r *Repo) Get(ctx, key K) (V, error)  // method form
//
// A positive detection stamps the standard three keys on the
// callable's meta bag (via the umbrella shape plugin):
//
//	shape            = "reader"
//	shape.key_type   = qualified type of `key`
//	shape.value_type = qualified type of `V`
//
// Consumers register the [Detector] alongside other shape
// detectors when constructing the umbrella plugin:
//
//	pipe.Use(shape.New().Detectors(reader.Detector()))
//
// Detector ordering matters when multiple shapes share a
// signature ([shape.New] honours registration order); reader is
// the canonical fallback for the "one key in, one value + error
// out" pattern and typically registers after more specific
// detectors (e.g. Deleter, Paginator) that would otherwise be
// shadowed.
package reader
