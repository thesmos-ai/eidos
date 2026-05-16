// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package nilsafe recognises the nil-safe mixin — the assertion
// that the annotated callable tolerates nil inputs and returns
// a defined result (typically the zero value plus a sentinel
// error) rather than panicking.
//
// The recognised directive is:
//
//	//+gen:mixin nilsafe
package nilsafe
