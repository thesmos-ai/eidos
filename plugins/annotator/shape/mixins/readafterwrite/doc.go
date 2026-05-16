// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package readafterwrite recognises the read-after-write mixin —
// the assertion that, immediately after a write to the named
// writer callable, a read returns the written value (linearisable
// from the writer's perspective).
//
// The recognised directive is:
//
//	//+gen:mixin readafterwrite write=Save
//
// Exposes a `write` parameter naming the paired writer callable.
// The mixin abstraction itself stores the value opaquely; the
// downstream test harness reads `shape.mixin.readafterwrite.write`
// to wire its assertion.
package readafterwrite
