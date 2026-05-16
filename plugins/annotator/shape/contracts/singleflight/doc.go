// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package singleflight recognises the get-or-compute contract — a
// single-role marker declaring that the annotated callable
// participates in singleflight de-duplication (concurrent calls
// with the same key share one in-flight computation).
//
// The recognised directive is:
//
//	//+gen:contract singleflight role=fn
package singleflight
