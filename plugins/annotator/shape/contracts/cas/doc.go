// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package cas recognises the compare-and-swap contract — a writer
// whose effect is conditional on a version field matching the
// caller's expectation. The directive carries a `version=` param
// naming the field; the resolver leaves the value as an opaque
// string (no callable lookup).
//
// The recognised directive is:
//
//	//+gen:contract cas role=writer version=Version
package cas
