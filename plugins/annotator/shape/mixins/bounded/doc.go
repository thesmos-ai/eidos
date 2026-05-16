// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package bounded recognises the bounded mixin — the assertion
// that the annotated callable's resource consumption stays
// within the declared upper bound (memory, queue depth, fanout).
//
// The recognised directive is:
//
//	//+gen:mixin bounded limit=100
package bounded
