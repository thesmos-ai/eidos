// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package ifmatch recognises the if-match conditional-writer
// contract — a writer that succeeds only when the existing
// record matches the supplied predicate. The `pred` param carries
// the predicate spelling (opaque).
//
// The recognised directive is:
//
//	//+gen:contract if-match role=writer pred=Version==Expected
package ifmatch
