// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

// Package cacheable recognises the cacheable mixin — the
// assertion that the annotated callable's result for a given
// input is stable over time, so callers may cache it
// indefinitely.
//
// The recognised directive is:
//
//	//+gen:mixin cacheable
package cacheable
