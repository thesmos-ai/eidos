// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package concurrent recognises the concurrent mixin — the
// assertion that the annotated callable is safe to invoke
// concurrently from multiple goroutines without external
// synchronisation.
//
// The recognised directive is:
//
//	//+gen:mixin concurrent
package concurrent
