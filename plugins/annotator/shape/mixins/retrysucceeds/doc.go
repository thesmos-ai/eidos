// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package retrysucceeds recognises the retry-succeeds mixin —
// the assertion that a transient failure followed by a retry
// converges to a successful outcome (the operation is naturally
// retryable).
//
// The recognised directive is:
//
//	//+gen:mixin retrysucceeds
package retrysucceeds
