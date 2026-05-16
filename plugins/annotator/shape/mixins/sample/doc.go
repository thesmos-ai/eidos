// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package sample recognises the sample mixin — the assertion
// that the annotated callable's test suite uses sampled
// fixtures rather than exhaustive enumeration (intended for
// callables whose input space is too large for full coverage).
//
// The recognised directive is:
//
//	//+gen:mixin sample
package sample
