// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package timeout recognises the timeout mixin — the assertion
// that the annotated callable respects a deadline and returns
// promptly when the supplied context expires. Carries a
// `duration` parameter naming the expected timeout (parsed
// downstream).
//
// The recognised directive is:
//
//	//+gen:mixin timeout duration=5s
package timeout
