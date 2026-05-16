// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package monotonic recognises the monotonic mixin — the
// assertion that successive invocations of the annotated callable
// observe a never-decreasing value (e.g. a counter, a version
// number, a high-water mark).
//
// The recognised directive is:
//
//	//+gen:mixin monotonic
//
// No parameters; presence is the entire signal.
package monotonic
