// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package ratelimit recognises the rate-limit contract — a
// callable bounded by a per-time-unit rate and an instantaneous
// burst capacity. Both params are opaque literals.
//
// The recognised directive is:
//
//	//+gen:contract rate-limit role=fn rate=100 burst=10
package ratelimit
