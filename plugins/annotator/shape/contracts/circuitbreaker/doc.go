// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package circuitbreaker recognises the circuit-breaker contract
// — a single-role marker declaring that the annotated callable
// wraps a downstream operation with circuit-breaker semantics
// (fail-fast after repeated failures).
//
// The recognised directive is:
//
//	//+gen:contract circuit-breaker role=fn
package circuitbreaker
