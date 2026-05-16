// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package streamreader recognises a streaming reader — a
// callable returning Go 1.23's iter.Seq[V] or iter.Seq2[V, error]
// as a range-over-func iterator.
//
// The recognised Go signatures are:
//
//	func (r *Repo) All(ctx context.Context) iter.Seq[V]
//	func (r *Repo) AllWithFilter(ctx context.Context, k K) iter.Seq[V]
//	func (r *Repo) All(ctx context.Context) iter.Seq2[V, error]
//
// A positive detection stamps:
//
//	shape                       = "streamreader"
//	shape.key_type              = qualified type of K (when an input key is present)
//	shape.value_type            = qualified type of V (the iterator element)
//	shape.streamreader.variant  = "seq" | "seq2"
package streamreader
